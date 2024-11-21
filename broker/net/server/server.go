package server

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"wasimoff/broker/net/server/cert"
)

// This Server is a simple wrapper for a http.Server server with optional TLS.
type Server struct {
	Http *http.Server
	cr   *cert.CertReloader
}

// Create a new Server with optional TLS using the CertReloader.
func NewServer(handler http.Handler, httpAddr, httpCert, httpKey string) (s *Server, err error) {

	// simple http/tls server
	s = &Server{
		Http: &http.Server{
			Addr:    httpAddr,
			Handler: handler,
		},
	}

	// maybe load a tls config for the server
	if httpCert != "" || httpKey != "" {
		s.cr, err = cert.NewCertReloader(httpCert, httpKey)
		if err != nil {
			return nil, fmt.Errorf("cannot load tls keypair: %w", err)
		}
		s.Http.TLSConfig = s.cr.GetTLSConfig()
	}

	return
}

// Addr returns the base listening address, like https?://host:port
func (s *Server) Addr() string {
	protocol := "http"
	if s.Http.TLSConfig != nil {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s", protocol, s.Http.Addr)
}

func (s *Server) ListenAndServe() error {

	// signal handler to close connections on CTRL-C
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	// start the HTTP server in background, with a channel for errors
	httpErr := make(chan error)
	go func() {
		if s.Http.TLSConfig != nil {
			httpErr <- s.Http.ListenAndServeTLS("", "")
		} else {
			httpErr <- s.Http.ListenAndServe()
		}
	}()

	// select the first signal and close server
	select {

	case <-sigint: // ^C pressed
		s.Http.Close()
		return fmt.Errorf("SIGINT received")

	case err := <-httpErr: // http.Server failed
		return fmt.Errorf("http.Server failed: %w", err)
	}

}

// Healthz returns a simple HandlerFunc simply replying with "OK"
func Healthz() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK\n"))
	}
}
