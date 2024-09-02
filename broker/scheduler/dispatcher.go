package scheduler

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// The Dispatcher takes a task queue and a provider selector strategy and then
// decides which task to send to which provider for computation.
func Dispatcher(queue chan *Task, selector Scheduler) {
	for task := range queue {

		// each task is handled in a separate goroutine
		go func(task *Task) {

			// schedule the task with a provider
			// TODO: retry task here
			call, err := selector.Schedule(context.TODO(), task)
			if err != nil {
				task.Result.Error = proto.String(err.Error())
				task.Done()
				return
			}

			// wait for completion and mark as completed
			<-call.Done
			task.Done()

			// signal completion to measure throughput
			if task.Result.Error == nil {
				selector.TaskDone()
			}

		}(task)
	}
}
