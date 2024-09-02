package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/provider"
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

func (s *SimpleMatchSelector) selectCandidates(task *Task) (candidates []*provider.Provider, err error) {

	// check if all the files are either raw binaries or known refs
	// TODO: this should really happen in the dispatcher already
	err = errors.Join(err, s.checkFile(task.Args.Binary))
	err = errors.Join(err, s.checkFile(task.Args.Rootfs))
	if err != nil {
		return nil, err
	}

	// create a list of needed files to check with the providers
	requiredFiles := make([]string, 0, 2)
	if task.Args.Binary.Ref != nil {
		requiredFiles = append(requiredFiles, *task.Args.Binary.Ref)
	}
	if task.Args.Rootfs.Ref != nil {
		requiredFiles = append(requiredFiles, *task.Args.Rootfs.Ref)
	}

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
		log.Printf("Task %s/%d couldn't find a Provider which satisfies: %v", task.Job.RequestID, task.Args.Task.Index, requiredFiles)
		// err = fmt.Errorf("no suitable provider found which satisfies all requirements")
	}
	return

}

func (s *SimpleMatchSelector) Schedule(ctx context.Context, task *Task) (call *provider.PendingWasiCall, err error) {
	call = provider.NewPendingWasiCall(task.Args, task.Result)
	for {

		providers, err := s.selectCandidates(task)
		if err != nil {
			return nil, err
		}

		// wrap parent context in a short timeout
		timeout, cancel := context.WithTimeout(ctx, time.Second)

		// submit the task normally with new context
		err = dynamicSubmit(timeout, call, providers)
		if err != nil && ctx.Err() == nil && timeout.Err() == err {
			// parent context not cancelled and err == our timeout,
			// so reschedule in hopes of picking up changes in provider store
			cancel()
			continue // retry
		}
		cancel()
		return call, err

	}
}

func (s *SimpleMatchSelector) TaskDone() {
	s.store.RateTick()
}

func (s *SimpleMatchSelector) checkFile(f *pb.File) error {
	// TODO: move this check to a Storage method
	// can't both be nil
	if f.Blob == nil && f.Ref == nil {
		return fmt.Errorf("can't use this file: both blob and ref are nil")
	}
	// blob is given directly, ok
	if f.Blob != nil {
		return nil
	}
	// ref is given and a known sha256 in storage, ok
	ref := f.GetRef()
	if s.store.Storage.Files[ref] != nil {
		return nil
	}
	// ref is given and can be looked-up, ok
	ref, ok := s.store.Storage.Lookup[ref]
	if ok && s.store.Storage.Files[ref] != nil {
		f.Ref = &ref
		return nil
	}
	// couldn't resolve the file
	return fmt.Errorf("can't use this file: not found in storage")
}
