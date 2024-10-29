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
	termclear()
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

	// maybe register the pprof handler
	if conf.Debug {
		pprofHandler(mux)
		log.Printf("DEBUG: broker PID is %d", os.Getpid())
		log.Printf("DEBUG: pprof profiles at %s/debug/pprof/", broker.Addr())
	}

	// health message
	mux.HandleFunc(apiPrefix+"/healthz", server.Healthz())

	// create a provider store and scheduler
	store := provider.NewProviderStore(conf.FileStorage)
	// selector := scheduler.NewRoundRobinSelector(&store)
	// selector := scheduler.NewAnyFreeSelector(&store)
	selector := scheduler.NewSimpleMatchSelector(&store)

	// run request handler
	mux.HandleFunc(apiPrefix+"/run", scheduler.ExecHandler(&store, &selector, conf.Benchmode))

	// upload wasm binaries to providers
	mux.HandleFunc(apiPrefix+"/upload", scheduler.UploadHandler(&store))
	log.Printf("API routes at %s%s/{run,upload}", broker.Addr(), apiPrefix)

	// provider transports
	providerSocket := "/websocket/provider"
	mux.HandleFunc(providerSocket, provider.WebSocketHandler(broker, &store, conf.AllowedOrigins))
	log.Printf("Provider socket: %s%s", broker.Addr(), providerSocket)

	// Prometheus metrics
	if conf.Metrics {
		mux.Handle("/metrics", metrics.MetricsHandler(
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
		log.Printf("Prometheus metrics: %s%s", broker.Addr(), "/metrics")
	}

	// serve static files for frontend
	mux.Handle("/", http.FileServer(http.Dir(conf.StaticFiles)))

	// start listening http server
	log.Printf("Broker listening on %s", broker.Addr())
	if err := broker.ListenAndServe(); err != nil {
		log.Fatalf("oops: %s", err)
	}

}

// pprofHandler mimics what the net/http/pprof.init() does, but on a specified mux
func pprofHandler(mux *http.ServeMux) {
	// https://cs.opensource.google/go/go/+/refs/tags/go1.23.0:src/net/http/pprof/pprof.go;l=95
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}
