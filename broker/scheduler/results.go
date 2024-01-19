package scheduler

// The result of a task, which can be an error or some output.
type ExecResult struct {
	Error  string      `json:"error,omitempty"`
	Result *TaskResult `json:"result,omitempty"`
}

func FormatExecResults(tasks []*Task) []ExecResult {
	results := make([]ExecResult, len(tasks))
	for i, task := range tasks {
		if task.Result.Err != nil {
			results[i].Error = task.Result.Err.Error()
		} else {
			results[i].Result = task.Result
		}
	}
	return results
}
