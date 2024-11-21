package main

import (
	"log"
	"os"
	"slices"
	"text/tabwriter"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Configuration via environment variables with github.com/kelseyhightower/envconfig.
type Configuration struct {

	// HttpListen is the listening address for the HTTP server.
	HttpListen string `split_words:"true" default:"localhost:4080" desc:"Listening Addr for HTTP server"`

	// HttpCert and HttpKey are paths to a TLS keypair to optionally use for the HTTP server.
	// If none are given, a plaintext server is started. Reload keys with SIGHUP.
	HttpCert string `split_words:"true" desc:"Path to TLS certificate to use"`
	HttpKey  string `split_words:"true" desc:"Path to TLS key to use"`

	// AllowedOrigins is a list of allowed Origin headers for transport connections.
	AllowedOrigins []string `split_words:"true" desc:"List of allowed Origins for WebSocket"`

	// StaticFiles is a path with static files to serve; usually the webprovider frontend dist.
	StaticFiles string `split_words:"true" default:"../webprovider/dist/" desc:"Serve static files on \"/\" from here"`

	// FileStorage is a path to use for a persistent BoltDB database.
	// An empty string will use an ephemeral in-memory map[string]*File.
	FileStorage string `desc:"Use persistent BoltDB storage for files" default:":memory:"`

	// Activate the benchmarking mode where the Broker produces workload itself
	Benchmode bool `desc:"Activate benchmarking mode" default:"false"` // TODO

	// Expose metrics for Prometheus via /metrics
	Metrics bool `desc:"Enable Prometheus exporter on /metrics" default:"true"`

	// Enable the pprof handlers under /debug/pprof
	Debug bool `desc:"Enable /debug/pprof profile handlers" default:"false"`
}

// Prefix for envionment variable names.
const envprefix = "WASIMOFF"

//
//
//

// GetConfiguration checks if user requested help (-h/--help) and prints usage information
// or returns the configuration parsed from environment variables.
func GetConfiguration() (conf Configuration) {

	// print help if requested
	if len(os.Args) >= 2 && slices.ContainsFunc(os.Args[1:], func(arg string) bool {
		return arg == "-h" || arg == "--help"
	}) {
		tabs := tabwriter.NewWriter(os.Stdout, 1, 0, 4, ' ', 0)
		envconfig.Usagef(envprefix, &conf, tabs, usageHelpFormat)
		tabs.Flush()
		os.Exit(1)
	}

	// load .env file into environment
	if err := godotenv.Load(); err != nil {
		log.Fatalf("failed to load dotenv: %s", err)
	}

	// parse configuration from environment variables
	if err := envconfig.Process(envprefix, &conf); err != nil {
		log.Fatalf("failed parsing config: %s", err)
	}
	return
}

// see https://github.com/kelseyhightower/envconfig/blob/v1.4.0/usage.go#L31
const usageHelpFormat = `This application is configured with the following environment variables:
KEY	DESCRIPTION	DEFAULT
{{range .}}{{usage_key .}}	{{usage_description .}}	{{usage_default .}}
{{end}}`
