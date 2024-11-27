# ---> build the broker binary
FROM golang:1.23-bookworm AS broker

# compile the binary
WORKDIR /broker
COPY ./broker /broker
RUN CGO_ENABLED=0 go build -o broker


# ---> build the webprovider frontend dist
FROM node:22-bookworm AS frontend

# compile the frontend
WORKDIR /provider
COPY ./webprovider /provider
RUN yarn install && yarn build


# ---> build denoprovider for the terminal
# docker build --target provider -t wasimoff/provider .
#FROM denoland/deno:distroless-1.46.3 AS provider
#FROM denoland/deno:distroless-2.1.1 AS provider
FROM denoland/deno:distroless AS provider

# copy files
COPY ./denoprovider /app
COPY ./webprovider /webprovider
WORKDIR /app

# cache required dependencies
RUN ["deno", "cache", "main.ts"]

# launch configuration
ENTRYPOINT ["/tini", "--", "deno", "run", "--cached-only", \
  "--allow-env", "--allow-read=/app,/webprovider", "--allow-net", "main.ts"]


# ---> combine broker and frontend dist in default container
# docker build --target wasimoff -t wasimoff/broker .
FROM scratch AS wasimoff
COPY --from=broker   /broker/broker /broker
COPY --from=frontend /provider/dist /provider
ENTRYPOINT [ "/broker" ]

# :: minimum container configuration ::

# the TCP port to listen on with the HTTP server
ENV WASIMOFF_HTTP_LISTEN=":4080"

# filesystem path to frontend dist to be served
ENV WASIMOFF_STATIC_FILES="/provider"
