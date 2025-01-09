# get the git revision for version tags
REVISION := $(shell printf 'r%s-g%s' "$$(git rev-list --count HEAD)" "$$(git describe --always --abbrev=7 --match '^$$' --dirty)")

# build docker containers for broker and deno providers
.PHONY: broker provider
broker:
	docker build --target wasimoff -t ansemjo/wasimoff:$@-$(REVISION) .
	docker tag ansemjo/wasimoff:$@-$(REVISION) ansemjo/wasimoff:$@
provider:
	docker build --target provider -t ansemjo/wasimoff:$@-$(REVISION) .
	docker tag ansemjo/wasimoff:$@-$(REVISION) ansemjo/wasimoff:$@

# build the client binary
.PHONY: client
client: wasimoff
wasimoff: $(shell find client/ broker/ -name '*.go')
	go build -o $@ ./client/

# redeploy the wasimoff broker container on wasi.team
.PHONY: deploy
deploy: broker
	docker save ansemjo/wasimoff:broker | ssh wasiteam docker load
	ssh wasiteam "cd wasimoff/ && docker compose up -d broker"

# recompile the protobuf definitions
.PHONY: protoc protowatch
protoc: broker/net/pb/messages.pb.go webprovider/lib/proto/messages_pb.ts
protowatch: protoc
	inotifywait -m -e close_write messages.proto | while read null; do make protoc; done
broker/net/pb/messages.pb.go: messages.proto
	cd $(dir $@) && go generate
webprovider/lib/proto/messages_pb.ts: messages.proto
	cd webprovider/ && yarn run protoc
