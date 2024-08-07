package qhttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"
	"wasmoff/broker/qhttp/cert"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

// This Server is a wrapper for starting both an encrypted QUIC/WebTransport (UDP)
// and a plaintext HTTP (TCP) server (for proxying) with the same handlers.
type Server struct {
	Quic *webtransport.Server
	Http *http.Server
	cr   *cert.CertReloader
}

// Create a new Server. Internally just returns a `webtransport.Server` with some
// default settings for now.
func NewServer(handler http.Handler, httpAddr, quicAddr, quicCert, quicKey string, https bool) (*Server, error) {

	http.NewServeMux()

	// load a tls config for quic server
	cr, err := cert.NewCertReloader(quicCert, quicKey)
	if err != nil {
		return nil, fmt.Errorf("cannot load tls keypair: %w", err)
	}

	// quic/webtransport server
	q := &webtransport.Server{
		H3: http3.Server{
			Addr:            quicAddr,
			Handler:         handler,
			EnableDatagrams: false,
			QuicConfig: &quic.Config{
				MaxIdleTimeout:  5 * time.Second,
				KeepAlivePeriod: 2 * time.Second,
			},
			TLSConfig: cr.GetTLSConfig(),
		},
		//! this function allows *all* origins
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// http/tls server
	h := &http.Server{
		Addr: httpAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			q.H3.SetQuicHeaders(w.Header())
			handler.ServeHTTP(w, r)
		}),
	}
	if https {
		// reuse quic's tls config for the http server
		h.TLSConfig = cr.GetTLSConfig()
	}

	return &Server{q, h, cr}, nil
}

func (s *Server) ListenAndServe() error {

	// signal handler to close connections on CTRL-C
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	// an error channel for each server
	httpErr := make(chan error)
	quicErr := make(chan error)

	// start the HTTP server
	go func() {
		if s.Http.TLSConfig != nil {
			httpErr <- s.Http.ListenAndServeTLS("", "")
		} else {
			httpErr <- s.Http.ListenAndServe()
		}
	}()

	// start the QUIC/WebTransport server
	go func() { quicErr <- s.Quic.ListenAndServe() }()

	// select the first error and return it
	select {
	case <-sigint: // ^C pressed
		// TODO: how to terminate gracefully? transport.H3.CloseGracefully() doesn't seem to work^W^W^W^Wis a TODO
		// https://github.com/quic-go/quic-go/blob/594440b04c7f385ad85fa53914dd02ce809da2ca/http3/server.go#L696
		s.Quic.Close()
		s.Http.Close()
		return fmt.Errorf("SIGINT received")
	case err := <-httpErr: // Http failed
		s.Quic.Close()
		return fmt.Errorf("TLS server failed: %w", err)
	case err := <-quicErr: // Quic failed
		s.Http.Close()
		return fmt.Errorf("QUIC server failed: %w", err)
	}

}

// TransportConfigHandler returns a HandlerFunc replying with the URL and certificate hash for WebTransport connections
func (s *Server) TransportConfigHandler(transport string) func(http.ResponseWriter, *http.Request) {
	// payload type with json properties
	type confPayload struct {
		// URL for WebTransport connections
		Transport string `json:"transport"`
		// undefined certhash means the certificate must be trusted
		Certhash string `json:"certhash,omitempty"`
	}
	payload := confPayload{transport, ""}
	// add certhash when using a selfsigned certificate
	if s.cr.IsSelfsigned() {
		payload.Certhash = s.cr.Certhash()
	}
	// encode the payload for endpoint
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		w.Header().Set("access-control-allow-origin", "*") // TODO: proper CORS
		json.NewEncoder(w).Encode(payload)
	}
}

// Healthz returns a simple HandlerFunc simply replying with "OK"
func Healthz() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK\n"))
	}
}
