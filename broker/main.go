package main

import (
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"wasimoff/broker/metrics"
	"wasimoff/broker/net/server"
	"wasimoff/broker/provider"
	"wasimoff/broker/scheduler"
)

func main() {
	banner()

	// use configuration from environment variables
	conf := GetConfiguration()
	log.Printf("%#v", &conf)

	// create a new broker on a new http handler
	mux := http.NewServeMux()
	broker, err := server.NewServer(mux, conf.HttpListen, conf.HttpCert, conf.HttpKey)
	if err != nil {
		log.Fatalf("failed to start server: %s", err)
	}

	// create a provider store and scheduler
	store := provider.NewProviderStore(conf.FileStorage)
	selector := scheduler.NewSimpleMatchSelector(store)
	// selector := scheduler.NewRoundRobinSelector(store)
	// selector := scheduler.NewAnyFreeSelector(store)

	// provider transports
	mux.HandleFunc("/api/provider/ws", provider.WebSocketHandler(store, conf.AllowedOrigins))
	log.Printf("Provider socket: %s/api/provider/ws", broker.Addr())

	// storage: serve files from and upload into store storage
	mux.Handle("/api/storage/{filename}", store.Storage)
	mux.HandleFunc("/api/storage/upload", scheduler.UploadHandler(store))
	log.Printf("Storage at %s/api/storage/...", broker.Addr())

	// client offloading request handler
	mux.HandleFunc("/api/client/run", scheduler.ExecHandler(store, &selector, conf.Benchmode))
	log.Printf("Client API at %s/api/client/run", broker.Addr())
	mux.HandleFunc("/api/client/ws", scheduler.ClientSocketHandler(store))
	log.Printf("Client socket: %s/api/client/ws", broker.Addr())

	// health message
	mux.HandleFunc("/healthz", server.Healthz())

	// pprof endpoint for debugging
	if conf.Debug {
		pprofHandler(mux, "/debug/pprof")
		log.Printf("DEBUG: broker PID is %d", os.Getpid())
		log.Printf("DEBUG: pprof profiles at %s/debug/pprof", broker.Addr())
	}

	// prometheus metrics
	if conf.Metrics {
		prometheusHandler(mux, "/metrics", store)
		log.Printf("Prometheus metrics: %s/metrics", broker.Addr())
	}

	// serve static files for frontend
	mux.Handle("/", http.FileServer(http.Dir(conf.StaticFiles)))

	// start listening http server
	log.Printf("Broker listening on %s", broker.Addr())
	if err := broker.ListenAndServe(); err != nil {
		log.Fatalf("oops: %s", err)
	}

}

//
// ---

// pprofHandler mimics what the net/http/pprof.init() does, but on a specified mux
func pprofHandler(mux *http.ServeMux, prefix string) {
	// https://cs.opensource.google/go/go/+/refs/tags/go1.23.0:src/net/http/pprof/pprof.go;l=95
	mux.HandleFunc(prefix+"/", pprof.Index)
	mux.HandleFunc(prefix+"/cmdline", pprof.Cmdline)
	mux.HandleFunc(prefix+"/profile", pprof.Profile)
	mux.HandleFunc(prefix+"/symbol", pprof.Symbol)
	mux.HandleFunc(prefix+"/trace", pprof.Trace)

}

// metrics endpoint for Prometheus
func prometheusHandler(mux *http.ServeMux, prefix string, store *provider.ProviderStore) {
	mux.Handle(prefix, metrics.MetricsHandler(
		// I'd love to put these funcs into the metrics package but that leads to an import cycle
		// gaugeFunc for the providers
		func() float64 {
			return float64(store.Size())
		},
		// gaugeFunc for the workers
		func() (f float64) {
			sum := 0
			store.Range(func(addr string, provider *provider.Provider) bool {
				sum += provider.CurrentLimit()
				return true
			})
			return float64(sum)
		},
	))

}
