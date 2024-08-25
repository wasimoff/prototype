package provider

import (
	"wasimoff/broker/storage"

	"github.com/puzpuzpuz/xsync"
)

// ProviderStore holds the currently connected providers, safe for concurrent access.
// It also keeps the list of files known to the provider in memory.
type ProviderStore struct {

	// Providers are held in a sync.Map safe for concurrent access
	providers *xsync.MapOf[string, *Provider]

	// Storage holds the uploaded files in memory
	Storage storage.FileStorage
}

// NewProviderStore properly initializes the fields in the store
func NewProviderStore() ProviderStore {
	return ProviderStore{
		providers: xsync.NewMapOf[*Provider](),
		Storage:   storage.NewFileStorage(),
	}
}

// --------------- stub methods for sync.Map ---------------

// Add a Provider to the Map.
func (s *ProviderStore) Add(provider *Provider) {
	s.providers.Store(provider.Get(Address), provider)
}

// Remove a Provider from the Map.
func (s *ProviderStore) Remove(provider *Provider) {
	s.providers.Delete(provider.Get(Address))
}

// Size is the current size of the Map.
func (s *ProviderStore) Size() int {
	return s.providers.Size()
}

// Load a Provider from the Map by its address.
func (s *ProviderStore) Load(addr string) *Provider {
	p, ok := s.providers.Load(addr)
	if !ok {
		return nil
	}
	return p
}

// Range will iterate over all Providers in the Map and call the given function.
// If the function returns `false`, the iteration will stop. See xsync.Map.Range()
// for more usage notes and (lack of) guarantees.
func (s *ProviderStore) Range(f func(addr string, provider *Provider) bool) {
	s.providers.Range(f)
}

// Keys will simply return the current keys (Provider addresses) of the Map.
func (s *ProviderStore) Keys() []string {
	keys := make([]string, 0, s.Size())
	s.Range(func(addr string, _ *Provider) bool {
		keys = append(keys, addr)
		return true
	})
	return keys
}

// Values will return the current values (Providers) of the Map.
func (s *ProviderStore) Values() []*Provider {
	providers := make([]*Provider, 0, s.Size())
	s.Range(func(_ string, prov *Provider) bool {
		providers = append(providers, prov)
		return true
	})
	return providers
}
