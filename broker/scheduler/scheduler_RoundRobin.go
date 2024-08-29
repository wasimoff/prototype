package scheduler

import (
	"context"
	"fmt"
	"wasimoff/broker/provider"

	"golang.org/x/exp/slices"
)

// The RoundRobinSelector is a very simple implementation of a ProviderSelector,
// which simply yields one provider after the next without concerning itself
// with *any* conditions or capacity counts.
type RoundRobinSelector struct {
	store *provider.ProviderStore
	// the index used to get the next provider
	index int
}

// Create a new RoundRobinSelector given an existing ProviderStore.
func NewRoundRobinSelector(store *provider.ProviderStore) RoundRobinSelector {
	return RoundRobinSelector{store, -1} // will increment to 0 on first use
}

func (s *RoundRobinSelector) Ok() (err error) {
	if s.store.Size() == 0 {
		return fmt.Errorf("provider store is empty")
	}
	return
}

func (s *RoundRobinSelector) selectCandidates(task *Task) (candidates []*provider.Provider, err error) {
	// round-robin actually got *harder* since using a map for the store ...

	// if the list is empty, return nil
	if err = s.Ok(); err != nil {
		return
	}

	// collect keys and sort them to make sure the roundrobin uses a stable order
	keys := s.store.Keys()
	slices.Sort[[]string](keys)

	// increment the index with wrap-around
	s.index = (s.index + 1) % len(keys)

	// return provider at index
	candidates = []*provider.Provider{s.store.Load(keys[s.index])}
	if candidates[0] == nil {
		// key must have been deleted between .Keys() and .Load(); retry ..
		return s.selectCandidates(task)
	}
	return
}

func (s *RoundRobinSelector) Schedule(ctx context.Context, task *Task) (call *provider.PendingWasiCall, err error) {

	providers, err := s.selectCandidates(task)
	if err != nil {
		return nil, err
	} else if len(providers) != 1 {
		return nil, fmt.Errorf("RoundRobinSelector.Select() did not return exactly one Provider")
	}

	call, err = dynamicSubmit(ctx, requestFromTask(task), providers)
	return

}

func (s *RoundRobinSelector) TaskDone() {
	s.store.RateTick()
}
