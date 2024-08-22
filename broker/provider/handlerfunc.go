package provider

import (
	"log"
	"net/http"
	"slices"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/net/server"
	"wasimoff/broker/net/transport"
)

// WebSocketHandler returns a http.HandlerFunc to be used on a route that shall serve
// as an endpoint for Providers to connect to. This particular handler uses WebSocket
// transport with either Protobuf or JSON encoding, negotiated using subprotocol strings.
func WebSocketHandler(server *server.Server, store *ProviderStore, origins []string) http.HandlerFunc {

	// warn about wildcard origin pattern
	if slices.Contains(origins, "*") {
		log.Println("WARNING: you're using the wildcard pattern in AllowedOrigins!")
	}

	return func(w http.ResponseWriter, r *http.Request) {

		// upgrade the transport
		wst, err := transport.UpgradeToWebSocketTransport(w, r, origins)
		if err != nil {
			log.Printf("[%s] New Provider: upgrade failed: %s", r.RemoteAddr, err)
			return
		}
		msg := transport.NewMessengerInterface(wst)

		// setup the provider instance
		provider := NewProvider(msg, r)
		defer provider.Close()

		// TODO: replace this up-front pushing with on-demand fetching by Providers (needs scheduler change, too!)
		// get the list of available files on provider
		if err = provider.ListFiles(); err != nil {
			log.Printf("[%s] New Provider: %s", provider.Addr, err)
			return
		}
		// upload all known files to the provider
		for _, file := range store.Storage.Files {
			err = provider.Upload(file)
			if err != nil {
				log.Printf("[%s] New Provider: initial Upload failed: %q: %s", provider.Addr, file.Name, err)
				return
			}
		}

		// add to the available store
		store.Add(provider)
		defer store.Remove(provider)
		log.Printf("[%s] New Provider connected using WebSocket", r.RemoteAddr)

		// handle incoming event messages
		go provider.eventReceiver()

		// wait until the session ends to defer cleanup
		<-r.Context().Done()
		log.Printf("[%s] Session has died", provider.Addr)

	}
}

// eventReceiver loops over incoming messages to update info on the provider
func (p *Provider) eventReceiver() {
	for event := range p.messenger.Events() {
		switch ev := event.(type) {

		case *pb.GenericEvent:
			log.Printf("[%s] says: %s", p.Addr, ev.GetMessage())

		case *pb.ProviderInfo:
			// log.Printf("[%s] ProviderInfo:\n%s", p.Addr, prototext.Format(ev))
			if v := ev.GetName(); v != "" {
				p.Info.Name = v
			}
			if v := ev.GetPlatform(); v != "" {
				p.Info.Platform = v
			}
			if v := ev.GetUseragent(); v != "" {
				p.Info.UserAgent = v
			}
			pool := ev.GetPool()
			// TODO: set active tasks .. see below
			if pool != nil && pool.Concurrency != nil {
				p.limiter.SetLimit(int(*pool.Concurrency))
			}

		case *pb.ProviderResources:
			// log.Printf("[%s] ProviderResources:\n%s", p.Addr, prototext.Format(ev))
			// TODO: set active tasks
			// The problem is that you can't really "set" a semaphore, so possibly
			// need to switch to a manual atomic, when providers are allowed to receive
			// tasks from multiple sources and we can't track it ourselves anymore.
			if ev.Concurrency != nil {
				p.limiter.SetLimit(int(*ev.Concurrency))
			}

		default:
			log.Printf("[%s] WARN: unknown event: %s", p.Addr, event.ProtoReflect().Descriptor().FullName())

		}
	}

}
