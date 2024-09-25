package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
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

// Return the true RemoteAddr from a http.Request (when proxied)
// TODO: if not running behind a trusted proxy, this isn't safe as clients can put anything in these headers
func ProxiedAddr(req *http.Request) string {
	remotes := []string{
		req.Header.Get("x-real-ip"),
		req.Header.Get("x-forwarded-for"),
	}
	for _, addr := range remotes {
		if addr != "" {
			// enclose in brackets, if it's ipv6 with colons
			if strings.Contains(addr, ":") {
				addr = fmt.Sprintf("[%s]", addr)
			}
			// get the port from RemoteAddr
			if split := strings.Split(req.RemoteAddr, ":"); len(split) >= 2 {
				return fmt.Sprintf("%s:%s", addr, split[len(split)-1])
			}
			return addr
		}
	}
	return req.RemoteAddr
}
