package scheduler

import (
	"context"
	"reflect"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/provider"
)

// Scheduler is a generic interface which must be fulfilled by a concrete scheduler,
// i.e. the type that selects suitable providers given task information and submits the task.
type Scheduler interface {
	// Check if the selector is OK in general (e.g. there is at least one provider)
	Ok() error
	// The Schedule function tries to submit a Task to a suitable Provider's queue and returns the WasmTask struct
	Schedule(ctx context.Context, task *Task) (*provider.ExecuteWasiCall, error)
}

// dynamicSubmit uses `reflect.Select` to dynamically select a Provider to submit a task to.
// This uses the Providers' unbuffered Queue, so that a task can only be submitted to a Provider
// when it currently has free capacity, without needing to busy-loop and recheck capacity yourself.
// Based on StackOverflow answer by Dave C. on https://stackoverflow.com/a/32381409.
func dynamicSubmit(ctx context.Context, run *pb.ExecuteWasiArgs, providers []*provider.Provider) (*provider.ExecuteWasiCall, error) {

	// construct the submission
	call := provider.NewExecuteWasiCall(run)

	// setup select cases
	cases := make([]reflect.SelectCase, len(providers), len(providers)+1)
	for i, p := range providers {
		if p.Submit == nil {
			panic("provider does not have a queue")
		}
		cases[i].Chan = reflect.ValueOf(p.Submit)
		cases[i].Dir = reflect.SelectSend
		cases[i].Send = reflect.ValueOf(call)
	}

	// add context.Done as select case for timeout or cancellation
	if ctx != nil {
		cases = append(cases, reflect.SelectCase{
			Chan: reflect.ValueOf(ctx.Done()),
			Dir:  reflect.SelectRecv,
		})
	}

	// select one of the queues and return the WasmCall struct
	i, _, _ := reflect.Select(cases)
	if i == len(providers) {
		// index out of bounds for providers, so it must be the ctx.Done
		return nil, ctx.Err()
	}
	call.Provider = providers[i]
	return call, nil

}
