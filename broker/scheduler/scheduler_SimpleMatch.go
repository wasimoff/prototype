package scheduler

import (
	"context"
	"fmt"
	"time"
	"wasmoff/broker/provider"
)

// The SimpleMatchSelector is another simple implementation of a ProviderSelector,
// which simply yields the first available provider with the required files in its store.
type SimpleMatchSelector struct {
	store *provider.ProviderStore
}

// Create a new SimpleMatchSelector given an existing ProviderStore.
func NewSimpleMatchSelector(store *provider.ProviderStore) SimpleMatchSelector {
	return SimpleMatchSelector{store}
}

func (s *SimpleMatchSelector) Ok() (err error) {
	if s.store.Size() == 0 {
		return fmt.Errorf("provider store is empty")
	}
	return
}

func (s *SimpleMatchSelector) selectCandidates(task *Task) (candidates []*provider.Provider, err error) {

	// if the list is empty, return nil
	if err = s.Ok(); err != nil {
		return
	}

	// assemble a slice with all the necessary files for this task
	requiredFiles := []string{}
	requiredFiles = append(requiredFiles, task.Binary)
	requiredFiles = append(requiredFiles, task.LoadFs...)

	// find suitable candidates
	candidates = make([]*provider.Provider, 0, s.store.Size())
	s.store.Range(func(addr string, p *provider.Provider) bool {
		// check for files
		for _, file := range requiredFiles {
			if !p.Has(file) {
				// missing requirement, continue
				return true
			}
		}
		// append candidate
		candidates = append(candidates, p)
		return true
	})

	// no candidates found?
	if len(candidates) == 0 {
		err = fmt.Errorf("no suitable provider found which satisfies all requirements")
	}
	return

}

func (s *SimpleMatchSelector) Schedule(ctx context.Context, task *Task) (call *provider.WasmCall, err error) {
	for {

		providers, err := s.selectCandidates(task)
		if err != nil {
			return nil, err
		}

		// wrap parent context in a short timeout
		timeout, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		// submit the task normally with new context
		call, err = dynamicSubmit(timeout, requestFromTask(task), providers)
		if err != nil && ctx.Err() == nil && timeout.Err() == err {
			// parent context not cancelled and err == our timeout,
			// so reschedule in hopes of picking up changes in provider store
			continue // loop again
		}
		return call, err

	}
}
