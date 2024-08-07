package provider

import (
	"context"
	"fmt"
	"log"
	netrpc "net/rpc"
	"sync"
	"wasimoff/broker/msgprpc"
	"wasimoff/broker/storage"

	"github.com/marusama/semaphore/v2"
	"github.com/quic-go/webtransport-go"
)

// Provider is a single connection initiated by a computing provier, i.e. a browser window
type Provider struct {
	// basic webtransport state
	session *webtransport.Session
	closing bool

	// wrapped bidi streams for rpc and messages
	rpc     *netrpc.Client
	msgchan *MessageChannel

	// unbuffered channel to submit tasks; can be `nil` if nobody's listening
	Submit chan *WasmCall

	// resizeable semaphore to limit number of concurrent tasks
	limiter semaphore.Semaphore

	// Information about the provider. Be a good citizen and don't change it from outside.
	Info  ProviderInfo
	Addr  string
	files map[string]*storage.File
}

// ProviderInfo holds meta information about the provider, like platform and pool capacity
type ProviderInfo struct {
	// mutex for general locking while updating string fields
	mutex sync.RWMutex
	// miscellaneous information
	providerInfoMessage
	// worker pool information received from provider
	providerPoolInfoMessage
}

// Setup a new Provider instance from a WebTransport session
func NewProvider(session *webtransport.Session, useQueue bool) (*Provider, error) {

	// accept a bidirectional stream for messages
	msgstream, err := session.AcceptStream(session.Context())
	if err != nil {
		return nil, fmt.Errorf("failed waiting for message stream: %v", err)
	}
	msgchan := NewMessageChannel(msgstream)

	// open a channel of our own for rpc requests
	rpcstream, err := session.OpenStream()
	if err != nil {
		return nil, fmt.Errorf("failed to open rpc stream: %v", err)
	}
	// instantiate net/rpc messagepack codec on it
	rpcclient := netrpc.NewClientWithCodec(msgprpc.NewCodec(rpcstream))

	// construct the provider
	p := &Provider{
		session, // webtransport
		false,   // closed

		rpcclient, // netrpc
		msgchan,   // messages

		nil,              // must be setup by acceptTasks
		semaphore.New(0), // pool semaphore

		ProviderInfo{},                 // abstract info
		session.RemoteAddr().String(),  // remote address
		make(map[string]*storage.File), // known filesystem
	}

	// start listening on task channel
	if useQueue {
		go p.acceptTasks()
	}

	return p, nil
}

// Close resets the streams and closes the entire WebTransport session to this provider
func (p *Provider) Close() error {
	if p.closing {
		return nil
	}
	p.closing = true
	p.msgchan.Close()
	p.rpc.Close()
	// close(p.Submit) // doc says the receiver shouldn't close
	return p.session.CloseWithError(0, "closing connection")
}

// Get the currently running tasks according to the semaphore
func (p *Provider) CurrentTasks() int {
	return p.limiter.GetCount()
}

// Get the currently configured Limit in the task semaphore
func (p *Provider) CurrentLimit() int {
	return p.limiter.GetLimit()
}

// WasmCall represents an active WebAssembly task, similarly to the rpc.Call struct
type WasmCall struct {
	Provider *Provider      // the Provider this task is running on
	Request  *WasmRequest   // Run arguments to the RPC call
	Reply    *WasmResponse  // Received *Reply from the Provider
	Error    error          // error encountered during the call
	Done     chan *WasmCall // receives itself when request completes
}

// NewWasmTask creates a new task struct with just enough to hand off to the TaskQueue
func NewWasmTask(run *WasmRequest) (task *WasmCall) {
	return &WasmCall{
		Request: run,
		Done:    make(chan *WasmCall, 1),
	}
}

func (p *Provider) acceptTasks() {
	if p.Submit == nil {
		p.Submit = make(chan *WasmCall)
	}
	// close Provider connection when the listener dies
	defer p.Close()
	for {

		// acquire a semaphore before accepting a task
		//? possibly off-by-one because we acquire and hold a semaphore before we even get a task
		//? what if we acquire and then the pool gets shrinked before we receive a task ..
		_ = p.limiter.Acquire(context.TODO(), 1) // no context, no err for now

		// get the call details from channel
		call := <-p.Submit

		// the Done channel MUST NOT be nil
		if call.Done == nil {
			call.Error = fmt.Errorf("call.Done for %s is nil, nobody is listening for this result", call.Request.Id)
			log.Println(call.Error)
			p.limiter.Release(1)
			return
		}

		// the Request MUST NOT be nil, of course
		if call.Request == nil {
			call.Error = fmt.Errorf("call.Request is nil")
			call.Done <- call
			return
		}

		// fill in the fields in WasmCall before handing off to RPC
		call.Provider = p
		call.Error = nil
		call.Reply = new(WasmResponse)

		// call the RPC in a goroutine to wait for completion and release the semaphore
		go func() {
			call.Error = p.run(call.Request, call.Reply)
			p.limiter.Release(1)
			call.Done <- call
		}()

	}
}
