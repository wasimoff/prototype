package scheduler

import (
	"context"
	"wasimoff/broker/net/pb"

	"google.golang.org/protobuf/proto"
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
				// TODO
				// now := tracer.Now("broker: task rpc completed")
				// // concatenate broker and provider traces
				// for _, ev := range call.Result.Trace {
				// 	ev.Label = "provider: " + ev.Label
				// 	task.trace.Events = append(task.trace.Events, ev)
				// }
				// task.trace.Events = append(task.trace.Events, now)
				// call.Result.Trace = task.trace.Events
			}
			task.Result = FormatResult(call.Result, err)
			task.Done()

			// signal completion to measure throughput
			if task.Result.Err == nil {
				selector.TaskDone()
			}

		}(task)
	}
}

func requestFromTask(task *Task) *pb.ExecuteWasiArgs {
	return &pb.ExecuteWasiArgs{
		Task: &pb.TaskMetadata{
			Id:    proto.String(task.Run.RunID),
			Index: proto.Uint64(uint64(task.Index)),
		},
		Binary: &pb.Executable{Binary: &pb.Executable_Reference{
			Reference: task.Binary,
		}},
		Args:     task.Args,
		Envs:     task.Envs,
		Stdin:    []byte(task.Stdin),
		Loadfs:   task.LoadFs,
		Datafile: &task.Datafile,
		Trace:    &task.Trace,
	}
}

func FormatResult(result *pb.ExecuteWasiResult, err error) *TaskResult {
	if err != nil {
		return &TaskResult{
			Err: err,
		}
	} else {
		return &TaskResult{
			Status:   int(result.GetStatus()),
			Stdout:   string(result.GetStdout()),
			Stderr:   string(result.GetStderr()),
			Datafile: result.GetDatafile(),
			Trace:    nil, // TODO
		}
	}
}
