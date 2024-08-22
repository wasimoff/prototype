package provider

import (
	"fmt"
	"log"
	"net/http"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/net/transport"
	"wasimoff/broker/storage"

	"github.com/marusama/semaphore/v2"
)

// TODO: orphaned logging
// log.Printf("[%s] Updated Info: { platform: %q, useragent: %q }", p.Addr, p.Info.Platform, p.Info.UserAgent)
// log.Printf("[%s] Workers: { pool: %d / %d }", p.Addr, pool, max)

// --------------- types of expected messages ---------------

type providerInfo struct {
	Name      string // logging-friendly name
	Platform  string // like `navigator.platform` in the browser
	UserAgent string // like `navigator.useragent` in the browser
}

// Provider is a single connection initiated by a computing provider
type Provider struct {

	// messenger connection
	messenger *transport.Messenger
	request   *http.Request
	closing   bool

	// unbuffered channel to submit tasks; can be `nil` if nobody's listening
	Submit chan *PendingWasiCall

	// resizeable semaphore to limit number of concurrent tasks
	limiter semaphore.Semaphore

	// Information about the provider. Be a good citizen and don't change it from outside.
	// TODO: implement with a getter to enforce
	Info  providerInfo
	Addr  string
	files map[string]*storage.File
}

// Setup a new Provider instance from a given Messenger
func NewProvider(m *transport.Messenger, req *http.Request) *Provider {

	// construct the provider
	p := &Provider{
		messenger: m,
		request:   req,
		closing:   false,

		Submit:  nil,              // must be setup by acceptTasks
		limiter: semaphore.New(0), // pool semaphore

		Info:  providerInfo{},                 // abstract info
		Addr:  req.RemoteAddr,                 // remote address
		files: make(map[string]*storage.File), // known filesystem
	}

	// start listening on task channel
	go p.acceptTasks()

	return p
}

// Close closes the underlying messenger connection to this provider
func (p *Provider) Close() {
	if p.closing {
		// TODO: not concurrent-safe, does it need to be?
		return
	}
	p.closing = true
	p.messenger.Close()
}

// Get the currently running tasks according to the semaphore
func (p *Provider) CurrentTasks() int {
	return p.limiter.GetCount()
}

// Get the currently configured Limit in the task semaphore
func (p *Provider) CurrentLimit() int {
	return p.limiter.GetLimit()
}

// PendingWasiCall represents an asynchronous WebAssembly exec call
type PendingWasiCall struct {
	Request *pb.ExecuteWasiArgs   // arguments to the call
	Result  *pb.ExecuteWasiResult // response from the Provider
	Error   error                 // error encountered during the call
	Done    chan *PendingWasiCall // receives itself when request completes
}

// NewPendingWasiCall creates a new call struct for the Submit chan
func NewPendingWasiCall(run *pb.ExecuteWasiArgs) *PendingWasiCall {
	return &PendingWasiCall{
		Request: run,
		Result:  new(pb.ExecuteWasiResult),
		Done:    make(chan *PendingWasiCall, 1),
	}
}

// done signals on the channel that this call is complete
func (call *PendingWasiCall) done() *PendingWasiCall {
	select {
	case call.Done <- call: // ok
	default: // never block here
	}
	return call
}

func (p *Provider) acceptTasks() {
	if p.Submit == nil {
		p.Submit = make(chan *PendingWasiCall) // unbuffered by design
	}
	// close Provider connection when the listener dies
	defer p.Close()
	for {

		// acquire a semaphore before accepting a task
		//? possibly off-by-one because we acquire and hold a semaphore before we even get a task
		//? what if we acquire and then the pool gets shrinked before we receive a task ..
		_ = p.limiter.Acquire(p.request.Context(), 1) // no context, no err for now

		// receive call details from channel
		call := <-p.Submit

		// the Done channel MUST NOT be nil
		if call.Done == nil {
			call.Error = fmt.Errorf("call.Done is nil, nobody is listening for this result")
			log.Println(call.Provider.Addr, call.Error)
			p.limiter.Release(1)
			return
		}

		// the Request MUST NOT be nil, of course
		if call.Request == nil {
			call.Error = fmt.Errorf("call.Request is nil")
			call.Done <- call
			p.limiter.Release(1)
			continue
		}

		// the Result must be allocated, too
		if call.Result == nil {
			call.Error = fmt.Errorf("call.Result is nil")
			call.Done <- call
			p.limiter.Release(1)
			continue
		}

		// fill in the fields in WasmCall before handing off to RPC
		call.Provider = p
		call.Error = nil

		// call the RPC in a goroutine to wait for completion and release the semaphore
		go func() {
			call.Error = p.run(call.Request, call.Result)
			p.limiter.Release(1)
			call.Done <- call
		}()

	}
}
