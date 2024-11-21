package provider

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/net/transport"

	"github.com/marusama/semaphore/v2"
	"google.golang.org/protobuf/proto"
)

// Provider is a single connection initiated by a computing provider
type Provider struct {
	messenger *transport.Messenger // messenger connection to provider

	// cancellable lifetime context to signal closure upwards
	lifetime  transport.Lifetime
	closeOnce sync.Once

	// unbuffered channel to submit tasks; can be `nil` if nobody's listening
	Submit chan *AsyncWasiTask

	// resizeable semaphore to limit number of concurrent tasks
	limiter semaphore.Semaphore
	waiting bool

	// information about the provider, to be accessed with Get()
	info map[ProviderInfoKey]string

	// list of files known on this provider
	files map[string]struct{}
}

type ProviderInfoKey string

const (
	Name      ProviderInfoKey = "name"      // a unique name for identification
	Address   ProviderInfoKey = "address"   // remote address of transport conn
	UserAgent ProviderInfoKey = "useragent" // software and architecture info
)

// Setup a new Provider instance from a given Messenger
func NewProvider(messenger *transport.Messenger) *Provider {
	lifetime := transport.NewLifetime(context.TODO())

	// construct the provider
	provider := &Provider{
		messenger: messenger,
		lifetime:  lifetime,
		Submit:    nil, // must be setup by acceptTasks
		limiter:   semaphore.New(0),
		info:      make(map[ProviderInfoKey]string),
		files:     make(map[string]struct{}),
	}

	// set known information
	provider.info[Name] = messenger.Addr()
	provider.info[Address] = messenger.Addr()
	provider.info[UserAgent] = "unknown"

	// start listening on task channel
	go provider.acceptTasks()

	return provider
}

func (p *Provider) Get(key ProviderInfoKey) string {
	return p.info[key]
}

func (p *Provider) Waiting() bool {
	return p.waiting
}

// -------------------- closure -------------------- >>

// Returns the cause of the closure or nil if Provider isn't closed yet.
func (p *Provider) Err() error {
	return p.lifetime.Err()
}

// Returns a channel to listen for lifetime closure.
func (p *Provider) Closing() <-chan struct{} {
	return p.lifetime.Closing()
}

// Close closes the underlying messenger connection to this provider
func (p *Provider) Close(reason error) {
	if reason == nil {
		reason = transport.ErrLifetimeEnded
	}
	p.closeOnce.Do(func() {
		p.messenger.Close(fmt.Errorf("closed from Provider: %w", reason))
		p.lifetime.Cancel(reason)
	})
}

// -------------------- limiter -------------------- >>

// Get the currently running tasks according to the semaphore
func (p *Provider) CurrentTasks() int {
	return p.limiter.GetCount()
}

// Get the currently configured Limit in the task semaphore
func (p *Provider) CurrentLimit() int {
	return p.limiter.GetLimit()
}

// -------------------- task channel -------------------- >>

// AsyncWasiTask is a parametrized WebAssembly task from an offloading job that
// can be submitted to a Provider's Submit() channel.
// TODO: add additional context information about task requirements here
type AsyncWasiTask struct {
	Context  context.Context
	Args     *pb.ExecuteWasiRequest  // arguments to the call
	Response *pb.ExecuteWasiResponse // response from the Provider
	Error    error                   // error encountered during scheduling or RPC
	done     chan *AsyncWasiTask     // receives itself when request completes
}

// NewAsyncWasiTask creates a new call struct for a scheduler
func NewAsyncWasiTask(
	ctx context.Context,
	args *pb.ExecuteWasiRequest,
	res *pb.ExecuteWasiResponse,
	done chan *AsyncWasiTask,
) *AsyncWasiTask {
	if done == nil {
		done = make(chan *AsyncWasiTask, 1)
	}
	if cap(done) == 0 {
		log.Panic("wasiTask: done channel is unbuffered")
	}
	if ctx == nil {
		log.Panic("wasiTask: context is nil")
	}
	return &AsyncWasiTask{ctx, args, res, nil, done}
}

// Done signals on the channel that this call is complete
func (t *AsyncWasiTask) Done() *AsyncWasiTask {
	select {
	case t.done <- t: // ok
	default: // never block here
		// TODO: probably shouldn't do this; except first two uses for general errors, this is always called in a goroutine
		log.Printf("WARN: AsyncWasiTask %s was blocked when signalling Done", t.Args.Info.TaskID())
	}
	return t
}

// Accept tasks on an unbuffered channel to submit to the Provider. Channels can
// be used in a DynamicSubmit, so calls from many different sources can be efficiently
// distributed to multiple Providers.
func (p *Provider) acceptTasks() (err error) {

	// initialize the channel
	if p.Submit == nil {
		p.Submit = make(chan *AsyncWasiTask) // unbuffered by design
	}

	// close Provider if the loop exits
	defer p.Close(err)

	for {

		// acquire a semaphore before accepting a task
		//? off-by-one because we acquire and hold a semaphore before we even get a task
		if err = p.limiter.Acquire(p.lifetime.Context, 1); err != nil {
			// nobody to notify and nothing to free, just quit
			return err
		}
		p.waiting = true

		select {

		// Provider is closing, quit the loop
		case <-p.lifetime.Closing():
			return p.Err()

		// receive call details from channel
		case call := <-p.Submit:
			p.waiting = false
			// the Done channel MUST NEVER be nil
			if call.done == nil {
				panic("call.Done is nil, nobody is listening for this result")
			}
			// the Request and Result most not be nil
			if call.Args == nil || call.Response == nil {
				call.Error = fmt.Errorf("call.Request and call.Result must not be nil")
				call.Done()
				p.limiter.Release(1)
				continue
			}
			// the context is already cancelled
			if call.Context.Err() != nil {
				call.Error = call.Context.Err()
				call.Done()
				p.limiter.Release(1)
				continue
			}

			// run the Request in a goroutine asynchronously
			// TODO: avoid gofunc by using a second listener on a `chan *PendingCall`
			go func() {
				call.Error = p.run(call.Context, call.Args, call.Response)
				// send cancellation event if error is due to context
				// TODO: semaphore is released before worker is definitely terminated
				if errors.Is(call.Error, context.Canceled) {
					p.messenger.SendEvent(p.lifetime.Context, &pb.CancelTask{
						Info:   call.Args.Info,
						Reason: proto.String(context.Canceled.Error()),
					})
				}
				call.Done()
				p.limiter.Release(1)
			}()

		}

	}
}
