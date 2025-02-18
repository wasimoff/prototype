package provider

import (
	"context"
	"log"
	wasimoff "wasimoff/proto/v1"
)

// AsyncTask is an individual parametrized task from an offloading job that
// can be submitted to a Provider's Submit() channel.
type AsyncTask struct {
	Context  context.Context
	Request  *wasimoff.Task_Request  // the overall request with metadata, QoS and task parameters
	Response *wasimoff.Task_Response // response containing either an error or specific output
	Error    error                   // errors encountered internally during scheduling or RPC
	done     chan *AsyncTask         // received itself when complete
}

// NewAsyncTask creates a new call struct for a scheduler
func NewAsyncTask(
	ctx context.Context,
	args *wasimoff.Task_Request,
	res *wasimoff.Task_Response,
	done chan *AsyncTask,
) *AsyncTask {
	if done == nil {
		done = make(chan *AsyncTask, 1)
	}
	if cap(done) == 0 {
		log.Panic("AsyncTask: done channel is unbuffered")
	}
	if ctx == nil {
		log.Panic("AsyncTask: context is nil")
	}
	return &AsyncTask{ctx, args, res, nil, done}
}

// Done signals on the channel that this call is complete
func (t *AsyncTask) Done() *AsyncTask {
	// TODO: re-add a select to never block here?
	t.done <- t
	return t
}

// Intercept replaces the done channel with another channel and returns the previous channel
func (t *AsyncTask) Intercept(interceptingChannel chan *AsyncTask) chan *AsyncTask {
	previous := t.done
	t.done = interceptingChannel
	return previous
}

// DoneCapacity returns the capacity of the done channel
func (t *AsyncTask) DoneCapacity() int {
	return cap(t.done)
}
