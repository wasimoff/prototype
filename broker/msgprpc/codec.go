package msgprpc

import (
	"io"
	"net/rpc"
	"sync"

	"github.com/vmihailenco/msgpack/v5"
)

// This codec is heavily inspired by github.com/hashicorp/net-rpc-msgpackrpc
// but uses github.com/vmihailenco/msgpack an a messagepack encoder instead.

// The required interface for net/rpc from the documentation:
//   type ClientCodec interface {
//     WriteRequest(*Request, any) error
//     ReadResponseHeader(*Response) error
//     ReadResponseBody(any) error
//     Close() error
//   }

type MessagePackCodec struct {
	conn   io.ReadWriteCloser
	closed bool
	lock   sync.Mutex
	enc    msgpack.Encoder
	dec    msgpack.Decoder
}

func NewCodec(conn io.ReadWriteCloser) *MessagePackCodec {
	return &MessagePackCodec{
		// hold connection to be able to close it
		conn: conn,
		// new msgpack en/decoders on connection
		enc: *msgpack.NewEncoder(conn),
		dec: *msgpack.NewDecoder(conn),
	}
}

func (codec *MessagePackCodec) Close() error {

	// already closed? is a NOP
	if codec.closed {
		return nil
	}

	// mark ourselves closed and close the connection
	codec.closed = true
	return codec.conn.Close()

}

func (codec *MessagePackCodec) WriteRequest(req *rpc.Request, body any) error {

	// check if already closed
	if codec.closed {
		return io.EOF
	}

	// acquire a write lock
	codec.lock.Lock()
	defer codec.lock.Unlock()

	// write header
	if err := codec.enc.Encode(req); err != nil {
		return err
	}

	// write body
	return codec.enc.Encode(body)

}

func (codec *MessagePackCodec) ReadResponseHeader(header *rpc.Response) error {
	return codec.read(header)
}

func (codec *MessagePackCodec) ReadResponseBody(body any) error {

	// doc: ReadResponseBody may be called with a nil argument to force the body of the response to be read and then discarded.
	if body == nil {
		var devnull any
		return codec.read(&devnull)
	}
	return codec.read(body)

}

func (codec *MessagePackCodec) read(v any) error {

	// check if already closed
	if codec.closed {
		return io.EOF
	}

	// try to decode the object
	return codec.dec.Decode(v)

}
