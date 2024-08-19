package provider

import (
	"log"
	"net/http"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/net/server"
	"wasimoff/broker/net/transport"
)

func WebSocketHandler(server *server.Server, store *ProviderStore, origins []string) http.HandlerFunc {

	// warn about wildcard origin pattern
	if len(origins) == 1 && origins[0] == "*" {
		log.Println("WARNING: you're using the wildcard pattern in allowed origins!")
	}

	return func(w http.ResponseWriter, r *http.Request) {

		// upgrade the transport
		log.Printf("[%s] New WebSocket Provider connection", r.RemoteAddr)
		m, err := transport.WrapMessengerInterface(transport.UpgradeToWebSocketTransport(w, r, origins))
		if err != nil {
			log.Printf("[%s] connection upgrade failed: %s", r.RemoteAddr, err)
			http.Error(w, "upgrade failed", http.StatusBadRequest)
			return
		}

		// setup the provider instance
		provider := NewProvider(m, r)
		defer provider.Close()
		log.Printf("[%s] Initialized Provider struct", provider.Addr)

		// get the list of available files on provider
		if err = provider.ListFiles(); err != nil {
			log.Printf("[%s] Failed fetching list of files: %s", provider.Addr, err)
			return
		}
		files := make([]string, 0, len(provider.files))
		for k := range provider.files {
			files = append(files, k)
		}
		log.Printf("[%s] Files: %q", provider.Addr, files)

		// upload all known files to the provider
		for _, file := range store.Storage.Files {
			err = provider.Upload(file)
			if err != nil {
				log.Printf("[%s] Failed uploading file %q: %s", provider.Addr, file.Name, err)
				return
			}
		}

		// add to the available store
		store.Add(provider)
		defer store.Remove(provider)
		log.Printf("[%s] Added to ProviderStore", provider.Addr)

		// handle incoming event messages
		go provider.RunEventHandler()

		// wait until the session ends to defer cleanup
		<-r.Context().Done()
		log.Printf("[%s] Session has died\n", provider.Addr)

	}
}

func (p *Provider) RunEventHandler() {
	for event := range p.messenger.IncomingEvents() {
		switch event.Event.(type) {

		case *pb.Event_Generic:
			log.Printf("[%s] says: %s", p.Addr, event.GetGeneric().GetMessage())

		case *pb.Event_ProviderInfo:
			info := event.GetProviderInfo()
			log.Printf("[%s] new info: %#v", p.Addr, info)
			if v := info.GetName(); v != "" {
				p.Info.Name = v
			}
			if v := info.GetPlatform(); v != "" {
				p.Info.Platform = v
			}
			if v := info.GetUseragent(); v != "" {
				p.Info.UserAgent = v
			}
			pool := info.GetPool()
			if pool != nil && pool.Concurrency != nil {
				p.limiter.SetLimit(int(*pool.Concurrency))
			}
			// TODO: tasks ..

		case *pb.Event_ProviderResources:
			info := event.GetProviderResources()
			log.Printf("[%s] new info: %#v", p.Addr, info)
			if info.Concurrency != nil {
				p.limiter.SetLimit(int(*info.Concurrency))
			}
			// TODO: problem is that you can't really "set" a semaphore
			// possibly I need to switch to a selfmanager atomic, when providers
			// are allowed to receive tasks from multiple sources

		default:
			log.Printf("[%s] unknown event received! %#v", p.Addr, event)

		}
	}

}
