package scheduler

import (
	"context"
	"fmt"
	"wasmoff/broker/provider"
	"wasmoff/broker/tracer"
)

// The Dispatcher takes a task queue and a provider selector strategy and then
// decides which task to send to which provider for computation.
func Dispatcher(queue chan *Task, selector Scheduler) {
	for task := range queue {
		go func(task *Task) {

			// schedule the task with a provider
			call, err := selector.Schedule(context.TODO(), task)
			if err != nil {
				task.Result = FormatResult(nil, err)
				task.Done()
				return
			}
			if task.trace != nil {
				task.trace.Now("broker: task scheduled")
			}

			// wait for completion, then write back the task result and mark as completed
			<-call.Done
			err = call.Error
			if task.trace != nil {
				now := tracer.Now("broker: task rpc completed")
				// concatenate broker and provider traces
				for _, ev := range call.Reply.Trace {
					ev.Label = "provider: " + ev.Label
					task.trace.Events = append(task.trace.Events, ev)
				}
				task.trace.Events = append(task.trace.Events, now)
				call.Reply.Trace = task.trace.Events
			}
			task.Result = FormatResult(call.Reply, err)
			task.Done()

		}(task)
	}
}

// TODO: why? apart from Id being a string the type is identical
func requestFromTask(task *Task) *provider.WasmRequest {
	return &provider.WasmRequest{
		Id:       fmt.Sprintf("%s/%04d", task.Run.RunID.String(), task.Index),
		Binary:   task.Binary,
		Args:     task.Args,
		Envs:     task.Envs,
		Stdin:    task.Stdin,
		Loadfs:   task.LoadFs,
		Datafile: task.Datafile,
		Trace:    task.Trace,
	}
}

func FormatResult(reply *provider.WasmResponse, err error) *TaskResult {
	if err != nil {
		return &TaskResult{Err: err}
	} else {
		return &TaskResult{
			Status:   reply.Status,
			Stdout:   reply.Stdout,
			Stderr:   reply.Stderr,
			Datafile: reply.Datafile,
			Trace:    reply.Trace,
		}
	}
}
