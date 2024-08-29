package provider

import (
	"context"
	"log"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/storage"

	"github.com/puzpuzpuz/xsync"
	"google.golang.org/protobuf/proto"
)

// ProviderStore holds the currently connected providers, safe for concurrent access.
// It also keeps the list of files known to the provider in memory.
type ProviderStore struct {

	// Providers are held in a sync.Map safe for concurrent access
	providers *xsync.MapOf[string, *Provider]

	// Storage holds the uploaded files in memory
	Storage storage.FileStorage

	// Broadcast is a channel to submit events for all Providers
	Broadcast chan proto.Message
}

// NewProviderStore properly initializes the fields in the store
func NewProviderStore() ProviderStore {
	store := ProviderStore{
		providers: xsync.NewMapOf[*Provider](),
		Storage:   storage.NewFileStorage(),
		Broadcast: make(chan proto.Message, 10),
	}
	go store.transmitter()
	return store
}

// ------------- broadcast events to everyone -------------

// transmitter forwards events from the chan to all Providers
func (s *ProviderStore) transmitter() {
	for event := range s.Broadcast {
		s.Range(func(_ string, p *Provider) bool {
			p.messenger.SendEvent(context.TODO(), event)
			return true
		})
	}
}

// --------------- stub methods for sync.Map ---------------

// Add a Provider to the Map.
func (s *ProviderStore) Add(provider *Provider) {
	s.providers.Store(provider.Get(Address), provider)
	log.Printf("ProviderStore: %d connected", s.Size())
	s.Broadcast <- &pb.ClusterInfo{Providers: proto.Uint32(uint32(s.Size()))}
}

// Remove a Provider from the Map.
func (s *ProviderStore) Remove(provider *Provider) {
	s.providers.Delete(provider.Get(Address))
	log.Printf("ProviderStore: %d connected", s.Size())
	s.Broadcast <- &pb.ClusterInfo{Providers: proto.Uint32(uint32(s.Size()))}
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
