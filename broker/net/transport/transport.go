package transport

import (
	"context"
	"errors"
	"wasimoff/broker/net/pb"
)

// Transport is an abstract connection interface which handles message-based
// wire serialization over the network for you.
type Transport interface {
	WriteMessage(context.Context, *pb.Envelope) error
	ReadMessage(context.Context, *pb.Envelope) error
	Addr() string // remote's address
	Close(cause error)
}

// Combining both network and serialization codec, any one of both parts can fail
// but a codec error is not necessarily a reason to close the connection.
var (
	ErrConnection = errors.New("transport connection error")
	ErrCodec      = errors.New("transport codec error")
)
