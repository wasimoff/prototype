package cert

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// CertReloader implements the pattern from https://stackoverflow.com/a/40883377,
// so that it can be reloaded during operation without stopping the server
type CertReloader struct {
	sync.RWMutex
	cert *tls.Certificate
	// keep file paths to reload
	certPath string
	keyPath  string
}

func NewCertReloader(certPath, keyPath string) (*CertReloader, error) {
	// instantiate by "reloading" for the first time
	cr := &CertReloader{certPath: certPath, keyPath: keyPath}
	if err := cr.reload(); err != nil {
		return nil, err
	}
	// add handlers to reload
	if !cr.IsSelfsigned() {
		// reload from filesystem on SIGHUP
		go func() {
			hup := make(chan os.Signal, 1)
			signal.Notify(hup, syscall.SIGHUP)
			for range hup {
				log.Printf("Received SIGHUP, reloading TLS keypair from %q and %q", cr.certPath, cr.keyPath)
				if err := cr.reload(); err != nil {
					log.Printf("ERR: failed TLS reload, keeping old keypair: %v", err)
				}
			}
		}()
	} else {
		log.Printf("using ephemeral keypair: %s", cr.Certhash())
		// periodically recreate ephemeral
		normaltick := 24 * time.Hour
		ticker := time.NewTicker(normaltick)
		go func() {
			for range ticker.C {
				if err := cr.reload(); err != nil {
					log.Printf("ERR: failed TLS reload, keeping old keypair: %v", err)
					ticker.Reset(time.Hour)
				} else {
					log.Printf("recreated ephemeral keypair: %s", cr.Certhash())
					ticker.Reset(normaltick)
				}
			}
		}()
	}
	return cr, nil
}

func (cr *CertReloader) reload() (err error) {
	var newcert tls.Certificate

	// load keypair from disk or generate ephemeral
	if cr.certPath != "" && cr.keyPath != "" {
		// both paths given, load from filesystem
		newcert, err = tls.LoadX509KeyPair(cr.certPath, cr.keyPath)
		if err != nil {
			return fmt.Errorf("failed loading keypair: %w", err)
		}
	} else if cr.certPath != "" || cr.keyPath != "" {
		// only one given, that's an error
		return fmt.Errorf("either none or both of (certPath, keyPath) must be set")
	} else {
		// none given, create a selfsigned certificate
		xcert, privkey, err := EphemeralWebTransportCert()
		if err != nil {
			return fmt.Errorf("failed generating keypair: %w", err)
		}
		newcert = tls.Certificate{
			Certificate: [][]byte{xcert.Raw},
			Leaf:        xcert,
			PrivateKey:  privkey,
		}
	}
	// replace the certificate
	cr.Lock()
	defer cr.Unlock()
	cr.cert = &newcert
	return nil
}

// IsSelfsigned returns if the cert is ephemeral or loaded from disk
func (cr *CertReloader) IsSelfsigned() bool {
	// no need for a separate bool in the struct
	return cr.keyPath == "" && cr.certPath == ""
}

// Certhash returns the hex-encoded sha256 hash of the certificate for use with the WebTransport constructor
func (cr *CertReloader) Certhash() string {
	sum := sha256.Sum256(cr.cert.Leaf.Raw)
	return hex.EncodeToString(sum[:])
}

// GetCertificateFunc retuns a function for use in `tls.Config.GetCertificate`
func (cr *CertReloader) GetCertificateFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		cr.RLock()
		defer cr.RUnlock()
		return cr.cert, nil
	}
}

// GetTLSConfig returns a `tls.Config`, which uses this reloader for its certificate
func (cr *CertReloader) GetTLSConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: cr.GetCertificateFunc(),
	}
}
