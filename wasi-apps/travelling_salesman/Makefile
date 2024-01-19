# probably shouldn't mix build environments (cargo + make) but it's a quicky way to an end ..

.PHONY: all native wasm static
all: native wasm static
native: tsp
wasm: tsp.wasm
static: tsp.static

# copy binaries from build directories
tsp: target/release/tsp
	cp -f $< $@
tsp.wasm: target/wasm32-wasi/release/tsp.opt.wasm
	cp -f $< $@
tsp.static: target/x86_64-unknown-linux-musl/release/tsp
	cp -f $< $@

# native binary
target/release/tsp: src/main.rs
	cargo build --release

# webassembly binary
target/wasm32-wasi/release/tsp.wasm: src/main.rs
	cargo build --target wasm32-wasi --release

# optimize wasm binary
target/wasm32-wasi/release/tsp.opt.wasm: target/wasm32-wasi/release/tsp.wasm
	wasm-opt -Oz -c $< -o $@

# native for another target, e.g. static musl
target/%/release/tsp: src/main.rs
	cargo build --target $* --release


# build the custom serverledge image
.PHONY: serverledge-image
serverledge-image: tsp.static Dockerfile wrapper.sh
	docker build -t ansemjo/serverledge-custom:tsp .

