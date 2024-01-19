package provider

import (
	"log"
	"net/http"
	"wasmoff/broker/qhttp"
)

func WebTransportHandler(server *qhttp.Server, store *ProviderStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// connection upgrade
		log.Printf("[%s] New QUIC connection", r.RemoteAddr)
		session, err := server.Quic.Upgrade(w, r)
		if err != nil {
			log.Printf("connection upgrade from %v failed: %s", r.RemoteAddr, err)
			w.WriteHeader(500)
			return
		}
		log.Printf("[%s] Successfully upgraded to WebTransport", r.RemoteAddr)

		// setup the provider instance
		provider, err := NewProvider(session, true)
		if err != nil {
			log.Printf("[%s] Provider init failed: %v", r.RemoteAddr, err)
			session.CloseWithError(0, "Provider init failed")
			return
		}
		defer provider.Close()
		log.Printf("[%s] Initialized Provider struct", provider.Addr)

		// send a simple ping-pong to make sure the rpc stream is open
		if err = provider.Ping(); err != nil {
			log.Printf("[%s] Ping failed: %s", provider.Addr, err)
		}

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

		// handle incoming messages
		go provider.RunMessageHandler()

		// wait until the session ends to defer cleanup
		<-session.Context().Done()
		log.Printf("[%s] Session has died\n", provider.Addr)
	}
}
