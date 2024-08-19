package scheduler

import (
	"log"
	"sync"
	"wasimoff/broker/tracer"
)

// TODO: the client-facing API should also use Protobuf definitions and then
// it will probably use the same messages as the Provider API, so they can
// be passed around. Otherwise, at least use (compatible) pointer types, so
// you can get away with just one allocation when "converting" structs.

// A single Task struct that is parametrized from the run configuration's Exec array.
type Task struct {
	// A reference to the initiating run configuration.
	Run *RunConfiguration `json:"-"`
	// The run index within the overall configuration.
	Index int `json:"index"`
	// The filename of the WASM binary.
	Binary string `json:"bin"`
	// Commandline arguments.
	Args []string `json:"args"`
	// Environment variables.
	Envs []string `json:"envs"`
	// List of OPFS files to "load" into the filesystem before execution.
	LoadFs []string
	// A certain file that should be returned with results.
	Datafile string
	// Text data should be sent to the process on stdin.
	Stdin string `json:"-"`
	// Pointer to be filled with the task's result.
	Result *TaskResult `json:"result"`
	// Signal the WaitGroup with a result.
	Done func() `json:"-"`
	// Enable timestamp tracing of this task.
	Trace bool `json:"trace"`
	trace *tracer.Trace
}

// The result of a task, which can be an error or some output.
type TaskResult struct {
	Err      error          `json:"error,omitempty"`
	Status   int            `json:"status"`
	Stdout   string         `json:"stdout"`
	Stderr   string         `json:"stderr"`
	Datafile []byte         `json:"datafile,omitempty"`
	Trace    []tracer.Event `json:"trace,omitempty"`
}

// DispatchTasks takes a run configuration, generates individual tasks from it,
// schedules them in the queue and eventually returns with the results of all
// those tasks.
func DispatchTasks(cfg *RunConfiguration, trace *tracer.Trace, ch chan *Task) []*Task {

	// create a waitgroup to await all tasks before returning result
	wg := new(sync.WaitGroup)

	// slice with all the queued tasks
	tasks := make([]*Task, 0, len(cfg.Exec))

	// iterate over individual items in exec
	for index, run := range cfg.Exec {

		// assemble the task
		task := Task{
			Run:      cfg,
			Index:    index,
			Binary:   cfg.Binary,
			Args:     run.Args,
			Envs:     append(run.Envs, cfg.Environ...),
			Stdin:    run.Stdin,
			Done:     wg.Done,
			LoadFs:   run.LoadFs,
			Datafile: run.Datafile,
			Trace:    cfg.Trace,
			trace:    trace,
		}
		tasks = append(tasks, &task)

		// queue it
		log.Printf("Queue task: %v", task)
		wg.Add(1)
		ch <- &task
		if trace != nil {
			trace.Now("broker: task queued")
		}
	}

	// wait for all tasks to finish
	wg.Wait()

	if trace != nil {
		trace.Now("broker: all tasks done")
	}

	return tasks
}
