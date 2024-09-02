package scheduler

import (
	"sync"
	"wasimoff/broker/net/pb"

	"google.golang.org/protobuf/proto"
)

// TODO: reintroduce tracer from commit 20c8978 and before

// Task is a parametrized task from an offloading job.
type Task struct {
	Job    *OffloadingJob // reference back to the originating job
	Args   *pb.ExecuteWasiArgs
	Result *pb.ExecuteWasiResponse
	Done   func() // signal the waitgroup for this job
}

// DispatchTasks takes a run configuration, generates individual tasks from it,
// schedules them in the queue and eventually returns with the results of all
// those tasks.
// TODO: accept a Context, so pending tasks can be cancelled from ExecHandler
func DispatchTasks(job *OffloadingJob, queue chan *Task) *pb.OffloadWasiJobResponse {

	// create a waitgroup to await all tasks before returning result
	wg := new(sync.WaitGroup)

	// common task metadata parent
	meta := func(i int) *pb.TaskMetadata {
		return &pb.TaskMetadata{
			Id:     &job.RequestID,
			Client: &job.ClientAddr,
			Index:  proto.Uint64(uint64(i)),
		}
	}

	// TODO: resolve binary and rootfs refs

	// create a slice with all the queued tasks
	parent := job.JobSpec.Common
	tasks := make([]*Task, 0, len(job.JobSpec.Tasks))
	for i, spec := range job.JobSpec.Tasks {

		// clone the parent to start
		taskArgs := pb.ExecuteWasiArgs{
			Task:      meta(i),
			Binary:    job.JobSpec.Binary,
			Args:      parent.Args,
			Envs:      parent.Envs,
			Stdin:     parent.Stdin,
			Rootfs:    parent.Rootfs,
			Artifacts: parent.Artifacts,
		}

		// override from specific task spec
		if spec.Args != nil {
			taskArgs.Args = spec.Args
		}
		if spec.Envs != nil {
			taskArgs.Envs = spec.Envs
		}
		if spec.Stdin != nil {
			taskArgs.Stdin = spec.Stdin
		}
		if spec.Rootfs != nil {
			taskArgs.Rootfs = spec.Rootfs
		}
		if spec.Artifacts != nil {
			taskArgs.Artifacts = spec.Artifacts
		}

		// create the task
		task := Task{
			Job:    job,
			Args:   &taskArgs,
			Result: &pb.ExecuteWasiResponse{},
			Done:   wg.Done,
		}

		// queue it
		wg.Add(1)
		queue <- &task
		tasks = append(tasks, &task)
	}

	// wait for all tasks to finish
	wg.Wait()

	response := &pb.OffloadWasiJobResponse{
		Results: make([]*pb.OffloadWasiTaskResult, len(tasks)),
	}
	for i, t := range tasks {
		response.Results[i] = (*pb.OffloadWasiTaskResult)(t.Result)
	}

	return response
}
