package main

import (
	"log"
	"net/http"
	"wasimoff/broker/net/server"
	"wasimoff/broker/provider"
	"wasimoff/broker/scheduler"
)

func main() {
	termclear()
	banner()

	// use configuration from environment variables
	conf := GetConfiguration()
	log.Printf("%#v", &conf)

	// create a new broker with default http handler
	broker, err := server.NewServer(http.DefaultServeMux, conf.HttpListen, conf.HttpCert, conf.HttpKey)
	if err != nil {
		log.Fatalf("failed to start server: %s", err)
	}

	// health message
	http.HandleFunc(apiPrefix+"/healthz", server.Healthz())

	// create a provider store and scheduler
	store := provider.NewProviderStore()
	// selector := scheduler.NewRoundRobinSelector(&store)
	// selector := scheduler.NewAnyFreeSelector(&store)
	selector := scheduler.NewSimpleMatchSelector(&store)

	// run request handler
	http.HandleFunc(apiPrefix+"/run", scheduler.ExecHandler(&selector, conf.Benchmode))

	// upload wasm binaries to providers
	http.HandleFunc(apiPrefix+"/upload", scheduler.UploadHandler(&store))

	// provider transports
	providerSocket := "/websocket/provider"
	http.HandleFunc(providerSocket, provider.WebSocketHandler(broker, &store, conf.AllowedOrigins))
	log.Println("Provider socket registered on", providerSocket)

	// serve static files for frontend
	http.Handle("/", http.FileServer(http.Dir(conf.StaticFiles)))

	// start listening http server
	httproto := "http"
	if broker.Http.TLSConfig != nil {
		httproto = "https"
	}
	log.Printf("Broker listening on %s://%s", httproto, conf.HttpListen)
	if err := broker.ListenAndServe(); err != nil {
		log.Fatalf("oops: %s", err)
	}

}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
