package main

// Configuration via environment variables with github.com/kelseyhightower/envconfig.
type Configuration struct {

	// HttpListen is the TCP port to listen on with the HTTP server
	HttpListen string `split_words:"true" default:":4080"`

	// HttpCert and HttpKey are paths to a TLS keypair to optionally use for the HTTP server
	HttpCert string `split_words:"true" default:""`
	HttpKey  string `split_words:"true" default:""`

	// AllowedOrigins is a list of allowed Origin headers for transport connections
	AllowedOrigins []string `split_words:"true" default:""`

	// StaticFiles is the path to the directory with the webprovider frontend dist
	StaticFiles string `split_words:"true" default:"../webprovider/dist/"`
}
