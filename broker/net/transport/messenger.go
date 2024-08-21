package transport

// This message endpoint is largely based on the net/rpc library in Go 1.22 and extends
// it for bidirectional usage with a predefined message type. Hence it is more like a
// message passing interface instead of "just" a generic RPC.
// See their LICENSE at https://cs.opensource.google/go/go/+/refs/tags/go1.22.4:LICENSE

// Copyright 2009 The Go Authors. All rights reserved.
// Modified 2024 Anton Semjonov

// TODO: handling of incoming requests (case pb.Envelope_Request) is not implemented yet

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/rpc"
	"reflect"
	"sync"
	"sync/atomic"
	"wasimoff/broker/net/pb"

	"google.golang.org/protobuf/proto"
)

var (
	ErrClosedTransport = fmt.Errorf("messenger transport is closed")
)

type Transport interface {
	// TODO: add context.Context to Read/Write?
	WriteMessage(*pb.Envelope) error
	ReadMessage(*pb.Envelope) error
	Addr() string
	Close() error
}

// Messenger is an abstraction over a Transport, which implements bidirectional
// RPC as well as simple Event messages.
type Messenger struct {
	transport      Transport          // the underlying transport
	incomingEvents chan proto.Message // channel for event messages

	txLock          sync.Mutex // only one sender
	eventSequence   atomic.Uint64
	requestSequence atomic.Uint64

	pendingLock sync.Mutex
	pending     map[uint64]*PendingCall
	closing     bool // client called Close
	stopping    bool // server told us to stop
}

// Create a new Messenger by wrapping a Transport, starting the handler for
// returning RPC responses and listening on incoming events. The caller needs
// to read from the channel returned by IncomingEvents() to receive events.
func NewMessengerInterface(transport Transport) *Messenger {
	m := &Messenger{
		transport:      transport,
		pending:        make(map[uint64]*PendingCall),
		incomingEvents: make(chan proto.Message, 10),
	}
	go m.receiver()
	return m
}

// Close the Messenger and underlying Transport.
// The receiver gofunc will tidy up pending requests.
func (m *Messenger) Close() {
	m.pendingLock.Lock()
	defer m.pendingLock.Unlock()
	if !m.closing {
		m.transport.Close()
		m.closing = true
	}
}

// Get a receive-only channel of incoming Events to handle.
func (m *Messenger) Events() <-chan proto.Message {
	return m.incomingEvents
}

// Write an event to the channel but never block when doing so!
func (m *Messenger) putEvent(event proto.Message) {
	select {
	case m.incomingEvents <- event: // ok
	default:
		log.Printf("WARN: receiver[%s]: dropped event, channel is full", m.transport.Addr())
	}
}

// The receiver will continuously read from the Transport and parse incoming
// messages. Responses are routed to their pending requests and Events are emitted
// on a channel. Incoming Requests are not implemented yet and are immediately
// responded to with an Error message. Call receiver in a gofunc after instantiation.
func (m *Messenger) receiver() {
	envelope := new(pb.Envelope)
	var receiveErr error
	for receiveErr == nil {

		// receive the next letter and switch by message type
		if receiveErr = m.transport.ReadMessage(envelope); receiveErr != nil {
			break
		}
		switch envelope.GetType() {

		case pb.Envelope_Request:
			// TODO: requests not implemented yet
			m.send(&pb.Envelope{
				Sequence: envelope.Sequence,
				Error:    proto.String("requests not implemented yet"),
			})
			continue

		case pb.Envelope_Event:
			// unpack event payload
			// TODO: type verification would
			event, err := envelope.Payload.UnmarshalNew()
			if err != nil {
				// this usually means that the message type is not known
				receiveErr = fmt.Errorf("unpacking event payload: %w", err)
				break
			}
			m.putEvent(event)
			continue

		case pb.Envelope_Response:
			// get the sequence number from message; valid RPC responses will never
			// be 0, which is the default if this field was not set in message
			seq := envelope.GetSequence()
			// fetch the pending call by sequence number
			m.pendingLock.Lock()
			call := m.pending[seq]
			delete(m.pending, seq)
			m.pendingLock.Unlock()
			if call == nil {
				// no such call was pending; either the sequence number was invalid or
				// the request partially failed upon sending
				log.Printf("WARN: receiver[%s]: no pending call for seq=%d", m.transport.Addr(), seq)
				continue
			}
			// check if this is a response error
			if envelope.Error != nil {
				call.Error = rpc.ServerError(envelope.GetError())
			}
			// unpack the payload into expected response
			if isNil(call.Response) {
				call.Error = fmt.Errorf("unpacking response payload: Response is nil")
				call.done()
				continue
			}
			err := envelope.Payload.UnmarshalTo(call.Response)
			// ignore payload err if this is an error response anyway
			if err != nil && call.Error == nil {
				call.Error = fmt.Errorf("unpacking response payload: %w", err)
			}
			call.done()

		default:
			receiveErr = fmt.Errorf("received an UNKNOWN message type")

		} // switch
	} // loop

	// when we got here, there was an error, so terminate any pending calls
	m.txLock.Lock()
	m.pendingLock.Lock()
	m.stopping = true
	if receiveErr == io.EOF {
		if m.closing {
			receiveErr = ErrClosedTransport
		} else {
			// TODO: probably makes more sense to have m.err instead of two bools for this case
			receiveErr = io.ErrUnexpectedEOF
			m.transport.Close()
			m.closing = true
		}
	}
	for _, call := range m.pending {
		call.Error = receiveErr
		call.done()
	}
	m.pendingLock.Unlock()
	m.txLock.Unlock()
}

// Send a prepared Envelope of some type on the transport.
func (m *Messenger) send(message *pb.Envelope) error {
	m.txLock.Lock()
	defer m.txLock.Unlock()
	if err := m.transport.WriteMessage(message); err != nil {
		return fmt.Errorf("failed send: %w", err)
	}
	return nil
}

// Send an Event using the next sequence number.
func (m *Messenger) SendEvent(event proto.Message) error {
	// get next sequence number
	seq := m.eventSequence.Add(1) // ++seq
	// pack and assemble the enveloped message
	payload, err := pb.Any(event)
	if err != nil {
		return fmt.Errorf("failed marshalling event: %w", err)
	}
	letter := &pb.Envelope{
		Sequence: proto.Uint64(seq),
		Type:     pb.Envelope_Event.Enum(),
		Payload:  payload,
	}
	// send over transport, return any errors directly
	return m.send(letter)
}

// Send a Request using the next sequence number and register a pending
// listener for the Response. Like the Go method in net/rpc.
func (m *Messenger) SendRequest(request proto.Message, response proto.Message, done chan *PendingCall) (call *PendingCall) {

	// ensure we have a buffered completion channel
	if done == nil {
		done = make(chan *PendingCall, 1) // TODO: net/rpc uses 10, why?
	} else {
		if cap(done) == 0 {
			log.Panic("done channel is unbuffered")
		}
	}

	// prepare the call struct
	call = &PendingCall{request, response, nil, done}
	// get next sequence number
	seq := m.requestSequence.Add(1) // ++seq
	// pack and assemble the enveloped message
	payload, err := pb.Any(request)
	if err != nil {
		call.Error = fmt.Errorf("failed marshalling request: %w", err)
		return call.done()
	}
	letter := &pb.Envelope{
		Sequence: proto.Uint64(seq),
		Type:     pb.Envelope_Request.Enum(),
		Payload:  payload,
	}

	// register this request in pending map
	m.pendingLock.Lock()
	if m.closing {
		m.pendingLock.Unlock()
		call.Error = ErrClosedTransport
		return call.done()
	}
	m.pending[seq] = call
	m.pendingLock.Unlock()

	// try sending over transport, unregister call on error
	if err := m.send(letter); err != nil {
		m.pendingLock.Lock()
		call = m.pending[seq]
		delete(m.pending, seq)
		m.pendingLock.Unlock()
		if call != nil {
			call.Error = err
			return call.done()
		}
	}
	return
}

// Send a Request synchronously by listening for completion directly.
func (m *Messenger) RequestSync(request, response proto.Message) error {
	// async call with a single-element channel and return its error directly
	call := <-m.SendRequest(request, response, make(chan *PendingCall, 1)).Done
	return call.Error
}

// PendingCall is used by Request to have something to write the response to.
type PendingCall struct {
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
