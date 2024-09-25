package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// current throughput
var Throughput = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "wasimoff_tasks_throughput",
	Help: "Current total throughput of successful tasks/second.",
})

func MetricsHandler(providerFunc, workerFunc func() float64) http.Handler {

	// number of connected providers
	connectedProviders := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "wasimoff_conn_providers",
		Help: "Currently connected Providers.",
	}, providerFunc)
	prometheus.MustRegister(connectedProviders)

	// total number of workers
	connectedWorkers := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "wasimoff_conn_providers_workers",
		Help: "Sum of Workers across currently connected Providers.",
	}, workerFunc)
	prometheus.MustRegister(connectedWorkers)

	// current throughput
	prometheus.MustRegister(Throughput)

	return promhttp.Handler()
}
