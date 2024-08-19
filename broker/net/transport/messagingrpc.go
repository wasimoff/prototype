package transport

// This message endpoint is largely based on the net/rpc library in Go 1.22 and extends
// it for bidirectional usage with a predefined message type. Hence it is more like a
// message passing interface instead of "just" a generic RPC.
// See their LICENSE at https://cs.opensource.google/go/go/+/refs/tags/go1.22.4:LICENSE

// Copyright 2009 The Go Authors. All rights reserved.
// Modified 2024 Anton Semjonov

// TODO: handling of incoming requests is not implemented at all yet

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"wasimoff/broker/net/pb"

	"google.golang.org/protobuf/proto"
)

// logging prefix for errors etc.
const prefix = "wasimoff/messages"

var (
	ErrClosedTransport = errors.New(prefix + ": Transport is closed")
	ErrNilResponse     = errors.New(prefix + ": response pointer is nil")
	ErrNotImplemented  = errors.New(prefix + ": not implemented yet")
)

// -------------------------------------------------------------------------------------------- //

type Transport interface {
	// TODO: add context.Context to Read/Write?
	WriteMessage(*pb.Envelope) error
	ReadMessage(*pb.Envelope) error
	Close() error
}

type Messenger struct {
	transport      Transport
	incomingEvents chan *pb.Event

	txLock          sync.Mutex
	eventSequence   atomic.Uint64
	requestSequence atomic.Uint64

	pendingLock sync.Mutex
	pending     map[uint64]*Call
	closing     bool // client called Close
	stopping    bool // server told us to stop
}

func NewMessengerInterface(transport Transport) *Messenger {
	m := &Messenger{
		transport:      transport,
		pending:        make(map[uint64]*Call),
		incomingEvents: make(chan *pb.Event, 10),
	}
	go m.receiver()
	return m
}

func WrapMessengerInterface(transport Transport, err error) (*Messenger, error) {
	if err != nil {
		return nil, err
	}
	return NewMessengerInterface(transport), nil
}

func (m *Messenger) Close() {
	m.pendingLock.Lock()
	defer m.pendingLock.Unlock()
	if !m.closing {
		m.transport.Close()
		m.closing = true
	}
}

func (m *Messenger) IncomingEvents() <-chan *pb.Event {
	return m.incomingEvents
}

func (m *Messenger) receiver() {
	var err error
	for err == nil {
		// receive into a fresh struct
		letter := new(pb.Envelope)
		if err = m.transport.ReadMessage(letter); err != nil {
			break
		}
		// switch by received message type first
		switch letter.Message.(type) {

		case *pb.Envelope_Request:
			// TODO: not implemented yet
			log.Printf("%s: unexpected request %d, not implemented yet", prefix, letter.GetSequence())
			m.send(&pb.Envelope{
				Sequence: letter.Sequence,
				Message: &pb.Envelope_Response{Response: &pb.Response{
					Error: proto.String("not implemented yet"),
				}},
			})
			// err = ErrNotImplemented

		case *pb.Envelope_Response:
			// get the sequence number from message; valid RPC responses will never
			// be 0, which is the default if this field was not set in message
			seq := letter.GetSequence()
			// fetch the pending call by sequence number
			m.pendingLock.Lock()
			call := m.pending[seq]
			delete(m.pending, seq)
			m.pendingLock.Unlock()
			if call == nil {
				// no such call was pending; either the sequence number was invalid or
				// the request partially failed upon sending
				log.Printf("WARN: receiver: no pending call for seq=%d", seq)
				continue
			}
			// get the actual response body
			response := letter.GetResponse()
			if response == nil {
				err = ErrNilResponse
				break
			}
			// no, this shouldn't close the connection completely
			// if response.Error != nil {
			// 	call.Error = rpc.ServerError(*response.Error)
			// }
			// TODO: switch to using google.protobuf.Any to use UnmarshalTo?
			// As implemented, the result is taken from the struct that is allocated for each
			// new imcoming message. I don't want to burden the consumer with unpacking the
			// entire pb.Envelope again but without using a double-pointer, there is no way to
			// efficiently put the result into a user-provided parameter.
			// Another consideration is number of allocations. Right now, the entire envelope
			// must be on the heap to be able to return parts of it "upwards". I am afraid that
			// the GC won't be able to collect the outer envelope as long as the inner response
			// is still in use. You could get around that with proto.Clone() or proto.Merge()
			// into a fresh heap-allocated struct. Then again, the envelope is not that much
			// larger than the request itself; probably better to avoid another allocation here.
			// Maybe proto.Merge() can also be a method to use stack-allocated memory that was
			// passed in from above but I doubt it, since the first thing the Response contains
			// is a pointer to the concrete oneof message type. The memory passed in will only
			// hold the memory for this first layer and a pointer. Or ... does Merge() do the
			// right thing if we already pass a Response with our expected concrete type struct?
			call.Response = response
			call.done()

		case *pb.Envelope_Event:
			// write events to channel but never block
			select {
			case m.incomingEvents <- letter.GetEvent(): // ok
			default:
				log.Printf("%s: dropped event %d, channel was full", prefix, letter.GetSequence())
			}

		}
	}
	// when we got here, there was an error, so terminate any pending calls
	m.txLock.Lock()
	m.pendingLock.Lock()
	m.stopping = true
	if err == io.EOF {
		if m.closing {
			err = ErrClosedTransport
		} else {
			// TODO: probably makes more sense to have m.err instead of two bools for this case
			err = io.ErrUnexpectedEOF
			m.transport.Close()
			m.closing = true
		}
	}
	for _, call := range m.pending {
		call.Error = err
		call.done()
	}
	m.pendingLock.Unlock()
	m.txLock.Unlock()
}

func (m *Messenger) send(message *pb.Envelope) error {
	//? TODO: does mutex make sense here, or move up to Event/Request?
	m.txLock.Lock()
	defer m.txLock.Unlock()
	if err := m.transport.WriteMessage(message); err != nil {
		return fmt.Errorf("%s: failed send: %w", prefix, err)
	}
	return nil
}

func (m *Messenger) SendEvent(event *pb.Event) error {

	// get next sequence number
	seq := m.eventSequence.Add(1)
	// assemble the enveloped message
	letter := &pb.Envelope{
		Sequence: proto.Uint64(seq),
		Message:  &pb.Envelope_Event{Event: event},
	}

	// send over transport, return any errors directly
	return m.send(letter)
}

func (m *Messenger) SendRaw(envelope *pb.Envelope) error {
	return m.send(envelope)
}

func (m *Messenger) SendRequest(request *pb.Request, done chan *Call) (call *Call) {

	// ensure we have a buffered completion channel
	if done == nil {
		done = make(chan *Call, 1) //? net/rpc uses 10, why?
	} else {
		if cap(done) == 0 {
			log.Panic(prefix + ": done channel is unbuffered")
		}
	}
	// prepare the call struct
	call = &Call{request, nil, nil, done}

	// get next sequence number
	seq := m.requestSequence.Add(1)
	// construct enveloped message
	letter := &pb.Envelope{
		Sequence: proto.Uint64(seq),
		Message:  &pb.Envelope_Request{Request: request},
	}

	// register this request in pending map
	m.pendingLock.Lock()
	if m.closing {
		m.pendingLock.Unlock()
		call.Error = ErrClosedTransport
		call.done()
		return
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
			call.done()
		}
	}
	// return the call either way, since any error is now enclosed within
	return
}

func (m *Messenger) RequestSync(request *pb.Request) (*pb.Response, error) {
	// async call with a single-element channel and return its error directly
	call := <-m.SendRequest(request, make(chan *Call, 1)).Done
	log.Printf("DEBUG RequestSync: err=%v, call=%#v", call.Error, call)
	return call.Response, call.Error
}

// Call is used by Request to have something to give the response to.
type Call struct {
	Request  *pb.Request  // sent request payload
	Response *pb.Response // decoded response payload
	Error    error        // general error
	Done     chan *Call   // receives *Call itself when it is complete
}

// done signals on the channel that this RPC is complete
func (call *Call) done() {
	select {
	case call.Done <- call: // ok
	default: // never block here
	}
}

// Identifier generates a random 128 bit / 16 byte value from system randomness. It
// was meant to be used as a safe replacement for uint64 sequence numbers. Not really
// needed, since the sequence counters for Request and Event are independent and do
// not need to be synchronized between client and server. Would be needed if you want
// "sessions", where a respondent can send more Requests in response to the same ID.
func Identifier() (id [16]byte) {
	// id = make([]byte, 16)
	if _, err := rand.Read(id[:]); err != nil {
		panic("failed to read randomness")
	}
	return
}
