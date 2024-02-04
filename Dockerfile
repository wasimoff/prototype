# ---> build the broker binary
FROM golang:1.21-bookworm as broker

# install more required software
#RUN apt update && apt install -y --no-install-recommends \

# compile the binary
WORKDIR /broker
COPY ./broker /broker
RUN CGO_ENABLED=0 go build -o broker


# ---> build the webprovider frontend dist
FROM node:20-bookworm as frontend

# compile the frontend
WORKDIR /provider
COPY ./webprovider /provider
RUN yarn install && yarn build


# ---> prepare an image with headless chromium as provider
FROM alpine:latest as provider
RUN apk add --no-cache chromium
ENV WASIMOFF_BROKER="http://localhost:4080"
ENV WASIMOFF_WORKERS="max"
ENTRYPOINT chromium --headless=new --verbose --enable-logging=stderr \
  --disable-extensions --no-sandbox \
  --no-first-run --no-default-browser-check \
  --no-pings --in-process-gpu \
  "$WASIMOFF_BROKER/#autoconnect=yes&workers=$WASIMOFF_WORKERS"


# ---> combine broker and frontend dist
FROM alpine:latest as wasimoff
RUN apk add --no-cache curl
COPY --from=broker   /broker/broker /broker
COPY --from=frontend /provider/dist /provider
ENTRYPOINT [ "/broker" ]

# :: configuration ::

# the TCP port to listen on with the plaintext HTTP server
ENV WASIMOFF_HTTP_LISTEN=":4080"

# the UDP port for the QUIC/WebTransport server
ENV WASIMOFF_QUIC_LISTEN=":4443"

# paths to certificate pair for the QUIC server, generate ephemeral if empty
ENV WASIMOFF_QUIC_CERT=
ENV WASIMOFF_QUIC_KEY=

# externally-reachable URL to the QUIC server
ENV WASIMOFF_TRANSPORT_URL="https://localhost:4443/transport"

# filesystem path to frontend dist to be served
ENV WASIMOFF_STATIC_FILES="/provider"
