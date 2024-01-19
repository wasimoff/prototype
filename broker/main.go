package main

import (
	"log"
	"net/http"
	"wasmoff/broker/provider"
	q "wasmoff/broker/qhttp"
	"wasmoff/broker/scheduler"

	"github.com/kelseyhightower/envconfig"
)

// common broker/v1 API prefix
const apiPrefix = "/api/broker/v1"

func main() {
	banner()

	// use configuration from environment variables
	var conf Configuration
	envconfig.MustProcess("wasimoff", &conf)
	log.Printf("%#v", conf)

	// create a new server with default http handler
	server, err := q.NewServer(http.DefaultServeMux, conf.HttpListen, conf.QuicListen, conf.QuicCert, conf.QuicKey)
	if err != nil {
		log.Fatalf("failed to start server: %s", err)
	}

	// simple health message
	http.HandleFunc(apiPrefix+"/healthz", q.Healthz())

	// create a provider store and scheduler
	store := provider.NewProviderStore()
	// selector := scheduler.NewRoundRobinSelector(&store)
	// selector := scheduler.NewAnyFreeSelector(&store)
	selector := scheduler.NewSimpleMatchSelector(&store)

	// run request handler
	http.HandleFunc(apiPrefix+"/run", scheduler.ExecHandler(&selector))

	// upload wasm binaries to providers
	http.HandleFunc(apiPrefix+"/upload", scheduler.UploadHandler(&store))

	// return configuration for webtransport connections
	http.HandleFunc(apiPrefix+"/config", server.TransportConfigHandler(conf.TransportURL))

	// webtransport endpoint to upgrade connection
	http.HandleFunc("/transport", provider.WebTransportHandler(server, &store))

	// serve static files for frontend
	http.Handle("/", http.FileServer(http.Dir(conf.StaticFiles)))

	// start listening on tls and quic/webtransport
	log.Printf("Server listening on http://%s (HTTP) / https://%s (QUIC)", conf.HttpListen, conf.QuicListen)
	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("oops: %s", err)
	}

}
