package provider

import (
	"context"
	"log"
	"time"
	"wasimoff/broker/metrics"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/storage"

	"github.com/paulbellamy/ratecounter"
	"github.com/puzpuzpuz/xsync"
	"google.golang.org/protobuf/proto"
)

// ProviderStore holds the currently connected providers, safe for concurrent access.
// It also keeps the list of files known to the provider in memory.
type ProviderStore struct {

	// Providers are held in a sync.Map safe for concurrent access
	providers *xsync.MapOf[string, *Provider]

	// Storage holds the uploaded files in memory
	Storage *storage.FileStorage

	// Broadcast is a channel to submit events for all Providers
	Broadcast chan proto.Message

	// ratecounter is used to keep track of throughput [tasks/s]
	ratecounter *ratecounter.RateCounter
}

// NewProviderStore properly initializes the fields in the store
func NewProviderStore(storagepath string) *ProviderStore {
	store := ProviderStore{
		providers:   xsync.NewMapOf[*Provider](),
		Broadcast:   make(chan proto.Message, 10),
		ratecounter: ratecounter.NewRateCounter(5 * time.Second),
	}
	if storagepath == "" || storagepath == ":memory:" {
		store.Storage = storage.NewMemoryFileStorage()
	} else {
		store.Storage = storage.NewBoltFileStorage(storagepath)
	}
	go store.transmitter()
	return &store
}

// ------------- broadcast events to everyone -------------

// transmitter forwards events from the chan to all Providers
func (s *ProviderStore) transmitter() {

	// send current throughput regularly
	go s.throughput(time.Second)

	// broadcast events from channel
	for event := range s.Broadcast {
		s.Range(func(_ string, p *Provider) bool {
			p.messenger.SendEvent(context.TODO(), event)
			return true
		})
	}

}

// -------------- ratecounter in tasks/second --------------

// RateTick should be called on successful Task completion to measure throughput
func (s *ProviderStore) RateTick() {
	s.ratecounter.Incr(1)
}

// throughput expects
func (s *ProviderStore) throughput(tick time.Duration) {
	for range time.Tick(tick) {
		tps := s.ratecounter.Rate() / 5
		// log.Println("Tasks/sec:", tps)

		// set gauge in metrics
		metrics.Throughput.Set(float64(tps))

		select {
		case s.Broadcast <- &pb.Throughput{
			Overall: proto.Float32(float32(tps)),
			// TODO: add individual contribution
		}:
			// ok
		default: // never block
		}

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
