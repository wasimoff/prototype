package transport

// This message endpoint is largely based on the net/rpc library in Go 1.22 and extends
// it for bidirectional usage with a predefined message type. Hence it is more like a
// message passing interface instead of "just" a generic RPC.
// See their LICENSE at https://cs.opensource.google/go/go/+/refs/tags/go1.22.4:LICENSE

// Copyright 2009 The Go Authors. All rights reserved.
// Modified 2024 Anton Semjonov

// TODO: handling of incoming requests (case pb.Envelope_Request) is not implemented yet

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"sync"
	"sync/atomic"
	wasimoff "wasimoff/proto/v1"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// Messenger is an abstraction over a Transport, which implements bidirectional
// RPC as well as simple Event messages.
type Messenger struct {
	transport Transport // the underlying transport
	lifetime  Lifetime  // cancellable long context

	events   chan proto.Message   // incoming event messages
	requests chan IncomingRequest // incoming request messages

	sendMutex       sync.Mutex        // only one sender
	envelope        wasimoff.Envelope // reusable for sending
	eventSequence   atomic.Uint64
	requestSequence atomic.Uint64

	pendingMutex sync.Mutex
	pending      map[uint64]*PendingCall
}

// Create a new Messenger by wrapping a Transport, starting the handler for
// returning RPC responses and listening on incoming events. The caller needs
// to read from the channel returned by Events() to receive events.
func NewMessengerInterface(transport Transport) *Messenger {

	// create a cancellable lifetime context to signal closure upwards
	lifetime := NewLifetime(context.TODO())

	// instantiate Messenger
	messenger := &Messenger{
		transport: transport,
		pending:   make(map[uint64]*PendingCall),
		events:    make(chan proto.Message, 32),
		requests:  make(chan IncomingRequest, 512),
		lifetime:  lifetime,
	}

	// start the receiver loop
	go messenger.receiver()

	return messenger
}

func (m *Messenger) Addr() string {
	return m.transport.Addr()
}

// -------------------- closure -------------------- >>

// Returns the cause of the closure or nil if Messenger isn't closed yet.
func (m *Messenger) Err() error {
	return m.lifetime.Err()
}

// Returns a channel to listen for lifetime closure.
func (m *Messenger) Closing() <-chan struct{} {
	return m.lifetime.Closing()
}

// Close the Messenger and underlying Transport.
// The receiver loop will tidy up pending requests.
func (m *Messenger) Close(reason error) {
	if reason == nil {
		reason = ErrLifetimeEnded
	}

	// prevent sending more requests
	m.pendingMutex.Lock()
	defer m.pendingMutex.Unlock()

	// close if not closed yet
	if m.Err() == nil {
		m.transport.Close(fmt.Errorf("closed from Messenger: %w", reason))
		m.lifetime.Cancel(reason)
		<-m.Closing()
		close(m.events)
	}
}

// -------------------- events -------------------- >>

// Get a receive-only channel of incoming Events to handle.
func (m *Messenger) Events() <-chan proto.Message {
	return m.events
}

// Write an incoming Event to the channel but never block when doing so!
func (m *Messenger) putEvent(event proto.Message) {
	select {
	case m.events <- event: // ok
	default:
		log.Printf("WARN: receiver[%s]: dropped event, channel is full", m.transport.Addr())
	}
}

// -------------------- requests -------------------- >>

// IncomingRequest holds information for the service to return a Response to.
type IncomingRequest struct {
	Seq     uint64
	Request proto.Message // received request payload
	Respond func(ctx context.Context, response proto.Message, err error) error
}

// Get a receive-only channel of incoming Events to handle.
func (m *Messenger) Requests() <-chan IncomingRequest {
	return m.requests
}

// Write an incoming Request to the channel but don't block when doing so!
func (m *Messenger) putRequest(seq uint64, request proto.Message) {
	r := IncomingRequest{
		Seq:     seq,
		Request: request,
		Respond: func(ctx context.Context, response proto.Message, err error) error {
			return m.SendResponse(ctx, seq, response, err)
		},
	}
	select {
	case m.requests <- r: // ok
	default:
		// can't immediately put it in the queue, spin up a goroutine for it
		go func(r IncomingRequest) {
			log.Printf("WARN: receiver[%s]: request %d, queue is full", m.transport.Addr(), r.Seq)
			m.requests <- r
		}(r)
	}
}

//

// -------------------- receiver -------------------- >>

// The receiver will continuously read from the Transport and parse incoming
// messages. Responses are routed to their pending requests and Events are emitted
// on a channel. Incoming Requests are not implemented yet and are immediately
// responded to with an Error message. Call receiver in a gofunc after instantiation.
func (m *Messenger) receiver() {
	var receiveErr error
	var envelope wasimoff.Envelope

	for receiveErr == nil {

		// receive the next letter and switch by message type
		if receiveErr = m.transport.ReadMessage(m.lifetime.Context, &envelope); receiveErr != nil {
			break
		}
		switch envelope.GetType() {

		case wasimoff.Envelope_Request:
			request, err := envelope.Payload.UnmarshalNew()
			if err != nil {
				// this usually means that the message type is not known
				receiveErr = fmt.Errorf("unpacking request payload: %w", err)
				break
			}
			m.putRequest(*envelope.Sequence, request)
			continue

		case wasimoff.Envelope_Event:
			// unpack event payload
			event, err := envelope.Payload.UnmarshalNew()
			if err != nil {
				// this usually means that the message type is not known
				receiveErr = fmt.Errorf("unpacking event payload: %w", err)
				break
			}
			m.putEvent(event)
			continue

		case wasimoff.Envelope_Response:
			// get the sequence number from message; valid RPC responses will never
			// be 0, which is the default if this field was not set in message
			seq := envelope.GetSequence()
			// fetch the pending call by sequence number
			call := m.popPending(seq)
			if call == nil {
				// no such call was pending; either the sequence number was invalid or
				// the request partially failed upon sending
				log.Printf("WARN: receiver[%s]: no pending call for seq=%d", m.transport.Addr(), seq)
				continue
			}
			// unpack the payload into expected response
			if envelope.Error != nil {
				call.Error = errors.New(*envelope.Error)
			} else {
				err := envelope.Payload.UnmarshalTo(call.Response)
				// ignore payload err if this is an error response anyway
				if err != nil && call.Error == nil {
					call.Error = fmt.Errorf("unpacking response payload: %w", err)
				}
			}
			call.done()
			continue

		default:
			receiveErr = fmt.Errorf("received an UNKNOWN message type")
			// break

		} // switch
	} // loop

	// receiver failed, tidy up
	m.sendMutex.Lock()
	m.pendingMutex.Lock()

	// if Close() was called, there will be a context cancellation in err
	err := m.Err()
	if errors.Is(err, context.Canceled) {
		m.transport.Close(err)
		receiveErr = err
	}
	// terminate any pending calls
	for _, call := range m.pending {
		call.Error = receiveErr
		call.done()
	}
	// set errors for future requests
	// TODO: reuse m.close() but it would currently deadlock because it also wants pendingMutex
	if m.Err() == nil {
		m.transport.Close(receiveErr)
		m.lifetime.Cancel(receiveErr)
		<-m.Closing()
		close(m.events)
	}
	m.pendingMutex.Unlock()
	m.sendMutex.Unlock()
}

// -------------------- transmitter -------------------- >>

// Send a prepared Envelope of some type on the transport.
func (m *Messenger) send(ctx context.Context, seq *uint64, mt *wasimoff.Envelope_MessageType, body proto.Message, reqErr error) (err error) {

	// pack the payload before locking
	var payload *anypb.Any
	if body != nil {
		payload, err = wasimoff.Any(body)
		if err != nil {
			return fmt.Errorf("failed marshalling payload: %w", err)
		}
	}

	// prevent concurrent access on envelope
	m.sendMutex.Lock()
	defer m.sendMutex.Unlock()
	m.envelope.Sequence = seq
	m.envelope.Type = mt
	if payload != nil {
		m.envelope.Payload = payload
	}
	if reqErr != nil {
		m.envelope.Error = proto.String(reqErr.Error())
	}

	// write the full message
	if werr := m.transport.WriteMessage(ctx, &m.envelope); werr != nil {
		return fmt.Errorf("failed send: %w", werr)
	}
	return nil
}

// Send a Response to a previous request.
func (m *Messenger) SendResponse(ctx context.Context, seq uint64, response proto.Message, err error) error {
	return m.send(ctx, &seq, wasimoff.Envelope_Response.Enum(), response, err)
}

// Send an Event using the next sequence number.
func (m *Messenger) SendEvent(ctx context.Context, event proto.Message) error {
	seq := m.eventSequence.Add(1) // ++seq
	return m.send(ctx, &seq, wasimoff.Envelope_Event.Enum(), event, nil)
}

// Send a Request using the next sequence number and register a pending
// listener for the Response. Like the Go method in net/rpc.
func (m *Messenger) SendRequest(ctx context.Context, request proto.Message, response proto.Message, done chan *PendingCall) *PendingCall {

	// ensure we have a buffered completion channel
	if done == nil {
		done = make(chan *PendingCall, 1) //?-- why does net/rpc use 10?
	} else {
		if cap(done) == 0 {
			log.Panic("done channel is unbuffered")
		}
	}

	// prepare the call struct, check if response isn't nil
	call := &PendingCall{ctx, request, response, nil, done}
	if response == nil || reflect.ValueOf(response).IsNil() {
		call.Error = fmt.Errorf("response interface is nil, refusing to send")
		return call.done()
	}

	// register this request in pending map
	seq := m.requestSequence.Add(1) // ++seq
	if err := m.addPending(seq, call); err != nil {
		// oops, we're closing, abort
		call.Error = fmt.Errorf("%w: %w", io.ErrClosedPipe, err)
		return call.done()
	}

	// send over transport
	if err := m.send(ctx, &seq, wasimoff.Envelope_Request.Enum(), request, nil); err != nil {
		// unregister call immediately on error
		if call = m.popPending(seq); call != nil {
			call.Error = err
			return call.done()
		}
	}
	return call
}

// Send a Request synchronously by listening for completion directly.
func (m *Messenger) RequestSync(ctx context.Context, request, response proto.Message) error {
	select {
	// async call with a single-element channel and return its error directly
	case call := <-m.SendRequest(ctx, request, response, make(chan *PendingCall, 1)).Done:
		return call.Error
	// context timeout or cancelled
	case <-ctx.Done():
		// TODO: remove pending call, need seq for that
		return ctx.Err()
	}
}

// -------------------- pending calls -------------------- >>

// PendingCall is used by Request to have something to write the response to.
type PendingCall struct {
	Context  context.Context
	Request  proto.Message     // sent request payload
	Response proto.Message     // decoded response payload
	Error    error             // general error
	Done     chan *PendingCall // receives *Call itself when it is complete
}

// done signals on the channel that this RPC is complete
func (call *PendingCall) done() *PendingCall {
	select {
	case call.Done <- call: // ok
	default: // never block here
	}
	return call
}

// register a pending call in the map, err if closing
func (m *Messenger) addPending(seq uint64, call *PendingCall) error {
	m.pendingMutex.Lock()
	defer m.pendingMutex.Unlock()
	if err := m.Err(); err != nil {
		return err
	}
	m.pending[seq] = call
	return nil
}

// load and delete a pending call from the map
func (m *Messenger) popPending(seq uint64) *PendingCall {
	m.pendingMutex.Lock()
	defer m.pendingMutex.Unlock()
	call := m.pending[seq]
	delete(m.pending, seq)
	return call
}

// -------------------- misc -------------------- >>

// Identifier generates a random 128 bit / 16 byte value from system randomness. It
// was meant to be used as a safe replacement for uint64 sequence numbers. Not really
// needed, since the sequence counters for Request and Event are independent and do
// not need to be synchronized between client and server. Would be needed if you want
// "sessions", where a respondent can send Requests of their own in response to a
// previous Request ID.
func Identifier() (id [16]byte) {
	// id = make([]byte, 16)
	if _, err := rand.Read(id[:]); err != nil {
		panic("failed to read randomness")
	}
	return
}

// isNil will check if i itself is nil or the value within the interface is nil. This is needed
// to check for call.Response fields because those are a proto.Message interface. Simple benchmarks
// show that is an order of magnitude slower than simple equals (1.94ns/op vs 0.245ns/op) though. :/
// Source: https://mangatmodi.medium.com/go-check-nil-interface-the-right-way-d142776edef1
func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(i).IsNil()
	}
	return false
}
