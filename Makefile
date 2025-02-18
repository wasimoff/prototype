# get the git revision for version tags
REVISION := $(shell printf 'r%s-g%s' "$$(git rev-list --count HEAD)" "$$(git describe --always --abbrev=7 --match '^$$' --dirty)")

.DEFAULT_GOAL := buf

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
.PHONY: buf bufwatch
buf:
	buf generate
bufwatch: buf
	inotifywait -m -e close_write messages.proto | while read null; do make buf; done
