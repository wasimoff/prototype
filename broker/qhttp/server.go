package qhttp

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

// This Server is a wrapper for starting both an encrypted QUIC/WebTransport (UDP)
// and a plaintext HTTP (TCP) server (for proxying) with the same handlers.
type Server struct {
	Quic *webtransport.Server
	Http *http.Server
	tls  *tlsConfig
}

// Holds the certificate for the QUIC server and whether
// it was selfsigned or loaded from an external file.
type tlsConfig struct {
	config     *tls.Config
	selfsigned bool
}

// Create a new Server. Internally just returns a `webtransport.Server` with some
// default settings for now.
func NewServer(handler http.Handler, httpAddr, quicAddr, quicCert, quicKey string, https bool) (*Server, error) {

	http.NewServeMux()

	// select tls config for quic server
	var tlsconf *tlsConfig
	if quicCert != "" && quicKey != "" {
		// load keys given in configuration parameters
		keypair, err := tls.LoadX509KeyPair(quicCert, quicKey)
		if err != nil {
			return nil, fmt.Errorf("can't load keypair: %w", err)
		}
		tlsconf = &tlsConfig{
			selfsigned: false,
			config: &tls.Config{
				Certificates: []tls.Certificate{keypair},
			},
		}
	} else if quicCert != "" || quicKey != "" {
		return nil, fmt.Errorf("either none or both of QuicCert + QuicKey must be given")
	} else {
		// generate a selfsigned certificate for quic/webtransport
		cfg, err := GetTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("can't generate selfsigned certificates: %w", err)
		}
		tlsconf = &tlsConfig{cfg, true}
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
			TLSConfig: tlsconf.config,
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
		h.TLSConfig = tlsconf.config
	}

	return &Server{q, h, tlsconf}, nil
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
	// encode certhash, if using a selfsigned certificate
	var certhash string
	if s.tls.selfsigned {
		sum := sha256.Sum256(s.tls.config.Certificates[0].Leaf.Raw)
		certhash = hex.EncodeToString(sum[:])
	}
	// payload type with json properties
	type confPayload struct {
		// URL for WebTransport connections
		Transport string `json:"transport"`
		// undefined certhash means the certificate must be trusted
		Certhash string `json:"certhash,omitempty"`
	}
	// encode the payload for endpoint
	payload := confPayload{transport, certhash}
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
