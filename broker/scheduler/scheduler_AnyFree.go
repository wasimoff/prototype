package scheduler

import (
	"context"
	"fmt"
	"wasimoff/broker/provider"
)

// The AnyFreeSelector is probably the simplest implementation of a ProviderSelector,
// which uses any free Provider without concerning itself with *any* task requrirements.
type AnyFreeSelector struct {
	store *provider.ProviderStore
}

// Create a new AnyFreeSelector given an existing ProviderStore.
func NewAnyFreeSelector(store *provider.ProviderStore) AnyFreeSelector {
	return AnyFreeSelector{store}
}

func (s *AnyFreeSelector) Ok() (err error) {
	if s.store.Size() == 0 {
		return fmt.Errorf("provider store is empty")
	}
	return
}

func (s *AnyFreeSelector) selectCandidates(task *Task) (candidates []*provider.Provider, err error) {

	// if the list is empty, return nil
	if err = s.Ok(); err != nil {
		return
	}

	// return all the providers ...
	return s.store.Values(), nil
}

func (s *AnyFreeSelector) Schedule(ctx context.Context, task *Task) (call *provider.PendingWasiCall, err error) {

	providers, err := s.selectCandidates(task)
	if err != nil {
		return nil, err
	}

	call, err = dynamicSubmit(ctx, requestFromTask(task), providers)
	return

}
