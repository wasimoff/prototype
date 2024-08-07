package main

import (
	"log"
	"net/http"
	"wasmoff/broker/net/server"
	"wasmoff/broker/provider"
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

	// create a new broker with default http handler
	broker, err := server.NewServer(http.DefaultServeMux, conf.HttpListen, conf.QuicListen, conf.QuicCert, conf.QuicKey, conf.Https)
	if err != nil {
		log.Fatalf("failed to start server: %s", err)
	}

	// simple health message
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

	// return configuration for webtransport connections
	http.HandleFunc(apiPrefix+"/config", broker.TransportConfigHandler(conf.TransportURL))

	// webtransport endpoint to upgrade connection
	http.HandleFunc("/transport", provider.WebTransportHandler(broker, &store))

	// serve static files for frontend
	http.Handle("/", http.FileServer(http.Dir(conf.StaticFiles)))

	// start listening on tls and quic/webtransport
	var httproto string
	if conf.Https {
		httproto = "https"
	} else {
		httproto = "http"
	}
	log.Printf("Broker listening on %s://%s (HTTP) / https://%s (QUIC)", httproto, conf.HttpListen, conf.QuicListen)
	if err := broker.ListenAndServe(); err != nil {
		log.Fatalf("oops: %s", err)
	}

}
