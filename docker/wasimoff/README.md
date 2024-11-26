# Containerized Wasimoff Deployment

**TODO: this README is outdated.**



This repository includes a multi-stage `Dockerfile`, which:
* compiles the `broker` binary in a `golang:1.21-bookworm` image,
* compiles the webprovider frontend dist in a `node:20-bookworm` image,
* copies both in a barebones `alpine` image to save space and
* prepares another image with a headless Chromium to act as the provider.

You can build the combined Broker + Frontend image as usual with `docker build` because it's the default target:

```
docker build -t wasimoff:broker ./
```

The headless provider image can be built by specifying its target. This image contains *no Wasimoff-specific code!* It's just a headless Chromium, which opens the web page hosted by the broker by default.

```
docker build -t wasimoff:provider --target provider ./
```

## Deployment

Unfortunately, due to incompatabilities in the TLS handling of the WebTransport constructor
(see details in [`broker/README`](broker/README.md#tls-certificate)), only two deployment
scenarios with these container images are universally useful:

* `localhost`-only deployment via `docker compose` with multiple providers on the same machine
* Broker reachable for other participants using trusted TLS certificate

Several APIs used in the webprovider *require* a secure context, so using ephemerally-generated
self-signed certificates is generally not useful when the broker is accessed from another machine.
Even using a Docker-internal hostname across network namespaces in a compose config will
**not** work.

On the other hand, only Firefox currently appears to check the browser's trust store on
WebTransport connections. So even a pre-generated self-signed certificate will not be widely
supported by participants. Thus, a certificate signed by a trusted CA is needed.

### Localhost Docker Compose

A `docker-compose.yaml` file is provided, which starts the combined broker in one container and a headless provider in another container. The containers use the host's networking namespace, so that you can directly issue `client.sh` commands using `BROKER=http://localhost:4080`. Copy the `provider01` section multiple times if you want to simulate multiple providers and run:

```
docker compose up --build
```

The config is a little tricky and is not really suitable for a public deployment:
* as mentioned above, a secure context is required, so using a service name in the URL will not work; the containers must be in the same namespace to use `localhost`
* using `network_mode: service:broker` to put the provider into the broker's networking namespace seemed to work on first glance but the WebTransport socket wouldn't connect
* using `networking_mode: host` with a **rootless** Docker installation worked for communication between both containers but made the broker inaccessible from the host machine
* so please, use a normal, *system*-Docker to start this compose file :)

### Public Deployment

For a public deployment (which can be in your faculty or in the cloud) you'll need:

* a server with a valid hostname, say `broker.example.com`, with Docker installed
* a certificate keypair, e.g. in `/etc/letsencrypt/live/broker.example.com/{fullchain,privkey}.pem`

First, copy the combined container image to the server; either build it from sources directly
on the server, transfer it with `docker save ... | ssh docker load` or pull it from `docker.io`:

```
docker pull docker.io/ansemjo/wasimoff:starless24-broker
```

Make sure ports 4443 TCP and UDP are open. Start the container image with:

```
docker run --name wasimoff-broker \
  -p 4443:4080 -p 4443:4443/udp \
  -e WASIMOFF_HTTP_LISTEN=":4080" \
  -e WASIMOFF_QUIC_LISTEN=":4443" \
  -v /etc/.../fullchain.pem:/fullchain.pem \
  -v /etc/.../privkey.pem:/privkey.pem \
  -e WASIMOFF_QUIC_CERT="/fullchain.pem" \
  -e WASIMOFF_QUIC_KEY="/privkey.pem" \
  -e WASIMOFF_HTTPS=true \
  -e WASIMOFF_TRANSPORT_URL="https://broker.example.com:4443/transport" \
  docker.io/ansemjo/wasimoff:starless24-broker
```

Now connect your participant's browsers to `https://broker.example.com:4443/` or
start the provider image on other machines with:

```
docker run --rm -it \
  -e WASIMOFF_BROKER="https://broker.example.com:4443/" \
  -e WASIMOFF_WORKERS="max" \
  ansemjo/wasimoff:starless24-provider
```

Depending on your firewall config, you may need to add `--network=host`.

Now upload binaries and offload tasks with [`client.sh`](client/):

```
BROKER="https://broker.example.com:4443" ./client.sh ...
```