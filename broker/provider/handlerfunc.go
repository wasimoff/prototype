package provider

import (
	"fmt"
	"log"
	"net/http"
	"slices"
	"time"
	"wasimoff/broker/net/transport"
	wasimoff "wasimoff/proto/v1"
)

// WebSocketHandler returns a http.HandlerFunc to be used on a route that shall serve
// as an endpoint for Providers to connect to. This particular handler uses WebSocket
// transport with either Protobuf or JSON encoding, negotiated using subprotocol strings.
func WebSocketHandler(store *ProviderStore, origins []string) http.HandlerFunc {

	// warn about wildcard origin pattern
	if slices.Contains(origins, "*") {
		log.Println("WARNING: you're using the wildcard pattern in AllowedOrigins!")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		addr := transport.ProxiedAddr(r)

		// upgrade the transport
		wst, err := transport.UpgradeToWebSocketTransport(w, r, origins)
		if err != nil {
			log.Printf("[%s] New Provider: upgrade failed: %s", addr, err)
			return
		}
		msg := transport.NewMessengerInterface(wst)

		// setup the provider instance
		provider := NewProvider(msg)
		defer provider.Close(nil)

		// handle incoming event messages
		go provider.eventTransmitter()

		// get the list of available files on provider
		if _, err = provider.ListFiles(); err != nil {
			log.Printf("[%s] New Provider: %s", addr, err)
			return
		}

		// add provider to the store
		log.Printf("[%s] New Provider connected using WebSocket", addr)
		store.Add(provider)
		defer store.Remove(provider)

		// wait until the session ends to defer cleanup
		select {
		case <-r.Context().Done():
		case <-msg.Closing():
		case <-provider.Closing():
		}
		log.Printf("[%s] Provider Session closed", addr)

	}
}

// eventTransmitter loops to receive incoming messages or send updates to the provider
func (p *Provider) eventTransmitter() {

	// create a ticker to send regular updates
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {

		// send regular updates
		case <-ticker.C:
			// TODO, this branch was used for ClusterInfo originally

		// reject incoming requests
		case request, ok := <-p.messenger.Requests():
			if !ok {
				return // channel is closing, quit
			}
			// log.Printf("[%s] Request %d: %s", p.messenger.Addr(), request.Seq, prototext.Format(request.Request))
			request.Respond(p.lifetime.Context, nil, fmt.Errorf("requests not supported on client socket"))

		// handle incoming events
		case event, ok := <-p.messenger.Events():
			if !ok {
				return // channel is closing, quit
			}
			switch ev := event.(type) {

			case *wasimoff.Event_GenericMessage:
				// generic text message
				log.Printf("[%s] says: %s", p.Get(Address), ev.GetMessage())

			case *wasimoff.Event_ProviderHello:
				// initial hello with platform information
				if v := ev.GetName(); v != "" {
					p.info[Name] = v
				}
				if v := ev.GetUseragent(); v != "" {
					p.info[UserAgent] = v
					log.Printf("[%s] UserAgent: %s", p.Get(Address), v)
				}

			case *wasimoff.Event_ProviderResources:
				// TODO: set active tasks
				// The problem is that you can't really "set" a semaphore, so possibly
				// need to switch to a manual atomic, when providers are allowed to receive
				// tasks from multiple sources and we can't track it ourselves anymore.
				if ev.Concurrency != nil {
					log.Printf("[%s] Workers: %d", p.Get(Address), *ev.Concurrency)
					p.limiter.SetLimit(int(*ev.Concurrency))
				}

			case *wasimoff.Event_FileSystemUpdate:
				// update about stored files on provider
				for _, file := range ev.GetAdded() {
					// first add
					p.files[file] = struct{}{}
				}
				for _, file := range ev.GetRemoved() {
					// then remove, i.e. err on _not_ having the file
					delete(p.files, file)
				}

			default:
				log.Printf("[%s] WARN: unknown event: %s", p.Get(Address), event.ProtoReflect().Descriptor().FullName())

			}

		}
	}

}
