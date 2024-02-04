# wasimoff Broker

This is the Broker component of the WebAssembly-based computation offloading
project wasimoff. It is written in Go and provides the necessary WebTransport
socket for Providers to connect and receive tasks.

### Run, watch and build

In the simplest case, you can just run with the defaults locally, which will start
a plaintext HTTP server on port 4080 and a TLS-secured QUIC/WebTransport socket
on port 4443:

```
go run ./
```

If you have installed [mitranim/gow](https://github.com/mitranim/gow), you can "watch"
for changes during development and automatically restart the server:

```
gow -s run ./
```

To build the binary, use `go build` as usual. To build a static binary use:

```
CGO_ENABLED=0 go build -o broker
```

### Configuration

Configuration is done through environment variables. In case this README is not up-to-date,
check the available options in `configuration.go`: the field names in the `Configuration`
struct are split into words, prepended with `WASIMOFF` and joined with underscores `_` to
compose the environment variable name.

| env | description |
| --- | ----------- |
| WASIMOFF_HTTP_LISTEN | the port to listen on with the HTTP server |
| WASIMOFF_QUIC_LISTEN | the port for the QUIC/WebTransport server |
| WASIMOFF_QUIC_{CERT,KEY} | paths to PEM-encoded certificate and key pair for the QUIC server (see notes below) |
| WASIMOFF_HTTPS | reuse the above certificates to enable TLS for the HTTP server, too |
| WASIMOFF_TRANSPORT_URL | externally-reachable URL to the QUIC server |
| WASIMOFF_STATIC_FILES | filesystem path to static files to be served (e.g. the Vue frontend) |


#### TLS Certificate

The QUIC/WebTransport server **must** be TLS-secured, therefore a certificate-key-pair
is required. You can either generate one externally (see `gencerts.sh`) and pass in their
filenames here or let the broker generate an emphemeral keypair on launch. This is
possible because WebTransport connections in the browser [can use the certificate hash](https://developer.mozilla.org/en-US/docs/Web/API/WebTransport/WebTransport#browser_compatibility)
to check validity instead of needing to trust the certificate chain. **However**,
this feature is currently only supported in Chromium-based browsers!
Firefox on the other hand also checks the browser's certificate trust store, so you
could add your own development CA.

For the best support between browsers, you must use a publicly trusted certificate;
e.g. one obtained through the ACME protocol from LetsEncrypt.

* leave blank, create ephemeral keys, use `serverCertificateHashes`
  * Chrome: ok; Firefox, Safari: **no**
* create selfsigned certificate, add CA in browser, add paths
  * Chrome: **no**; Firefox: ok; Safari: unknown
* publicly trusted certificate, add paths
  * all: ok

#### External URL and ports

For any kind of serious deployment you should consider using publicly trusted certificates
as mentioned above. But using publicly trusted certificates usually also means using a
publicly resolveable address. Set the reachable URL to your QUIC server with `WASIMOFF_TRANSPORT_URL`.

Fun fact: you can run the plaintext HTTP server behind an nginx proxy and use port 443 for
**both** the HTTP server in nginx and the QUIC server in the broker because QUIC listens
for UDP packets, while nginx listens for TCP packets. The browser trying to establish a
WebTransport connection will use the correct transport.