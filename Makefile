# get the git revision for version tags
REVISION := $(shell printf 'r%s-g%s' "$$(git rev-list --count HEAD)" "$$(git describe --always --abbrev=7 --match '^$$' --dirty)")

.PHONY: broker provider
broker:
	docker build --target wasimoff -t ansemjo/wasimoff:$@-$(REVISION) .
	docker tag ansemjo/wasimoff:$@-$(REVISION) ansemjo/wasimoff:$@
provider:
	docker build --target provider -t ansemjo/wasimoff:$@-$(REVISION) .
	docker tag ansemjo/wasimoff:$@-$(REVISION) ansemjo/wasimoff:$@

.PHONY: client
client: wasimoff
wasimoff: $(shell find client/ broker/ -name '*.go')
	go build -o $@ ./client/
