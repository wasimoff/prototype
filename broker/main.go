package main

import (
	"log"
	"net/http"
	"wasimoff/broker/net/server"
	"wasimoff/broker/provider"
	"wasimoff/broker/scheduler"

	"github.com/kelseyhightower/envconfig"
)

// common broker/v1 API prefix
const apiPrefix = "/api/broker/v1"

func main() {
	banner()

	// use configuration from environment variables
	var conf Configuration
	envconfig.MustProcess("wasimoff", &conf)
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
	http.HandleFunc(apiPrefix+"/run", scheduler.ExecHandler(&selector))

	// upload wasm binaries to providers
	http.HandleFunc(apiPrefix+"/upload", scheduler.UploadHandler(&store))

	// provider transports
	http.HandleFunc("/websocket/provider", provider.WebSocketHandler(broker, &store, conf.AllowedOrigins))

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
