package transport

import (
	"context"
	"fmt"
)

// The Lifetime interface is pretty much just a Context and its CancelFunc to not
// reinvent the wheel here. Storing a Context in a struct is usually considered bad
// practice but it checks all the boxes that we need for a struct to be able to
// signal closure to the outside world and be safe when cancelled multiple times.
// The name should be an indication that it's not meant for individual requests.
type Lifetime struct {

	// Context with a long lifetime, DO NOT USE for single requests if a more
	// suitable context is available
	Context context.Context

	// cancellation function, aka. Die(), Stop(), Close(), ...
	Cancel context.CancelCauseFunc
}

// Create a new long-running context to signal closure.
func NewLifetime(parent context.Context) Lifetime {
	ctx, cancel := context.WithCancelCause(parent)
	return Lifetime{ctx, cancel}
}

// Returns a channel to listen for lifetime closure.
func (c *Lifetime) Closing() <-chan struct{} {
	return c.Context.Done()
}

// Returns the cause of the closure or nil if it isn't closed yet.
func (c *Lifetime) Err() error {
	return context.Cause(c.Context)
}

// wrap the context.Canceled; check with errors.Is()
var ErrLifetimeEnded = fmt.Errorf("%w: lifetime ended", context.Canceled)
