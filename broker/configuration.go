package main

import (
	"log"
	"os"
	"slices"
	"text/tabwriter"

	"github.com/kelseyhightower/envconfig"
)

// Configuration via environment variables with github.com/kelseyhightower/envconfig.
type Configuration struct {

	// HttpListen is the listening address for the HTTP server
	HttpListen string `split_words:"true" default:":4080" desc:"Listening Addr for HTTP server"`

	// HttpCert and HttpKey are paths to a TLS keypair to optionally use for the HTTP server
	HttpCert string `split_words:"true" desc:"Path to TLS certificate to use"`
	HttpKey  string `split_words:"true" desc:"Path to TLS key to use"`

	// AllowedOrigins is a list of allowed Origin headers for transport connections
	AllowedOrigins []string `split_words:"true" desc:"List of allowed Origins for WebSocket"`

	// StaticFiles is the path to the directory with the webprovider frontend dist
	StaticFiles string `split_words:"true" default:"../webprovider/dist/" desc:"Serve static files on \"/\" from here"`
}

// GetConfiguration checks if user requested help (-h/--help) and prints usage information
// or returns the configuration parsed from environment variables.
func GetConfiguration() (conf Configuration) {

	// print help if requested
	if len(os.Args) >= 2 && slices.ContainsFunc(os.Args[1:], func(arg string) bool {
		return arg == "-h" || arg == "--help"
	}) {
		tabs := tabwriter.NewWriter(os.Stdout, 1, 0, 4, ' ', 0)
		envconfig.Usagef(envconfigPrefix, &conf, tabs, usageHelpFormat)
		tabs.Flush()
		os.Exit(1)
	}

	// parse configuration from environment variables
	if err := envconfig.Process(envconfigPrefix, &conf); err != nil {
		log.Fatalf("failed parsing config: %s", err)
	}
	return
}

// see https://github.com/kelseyhightower/envconfig/blob/v1.4.0/usage.go#L31
const usageHelpFormat = `This application is configured with the following environment variables:
KEY	DESCRIPTION	DEFAULT
{{range .}}{{usage_key .}}	{{usage_description .}}	{{usage_default .}}
{{end}}`
