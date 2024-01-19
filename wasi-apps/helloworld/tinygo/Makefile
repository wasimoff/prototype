exe.wasm: main.go
	tinygo build -target=wasi -o $@ $<

.PHONY: run-wasm run-native
run-wasm: exe.wasm
	wasmtime run --dir ./ -- $< argument

run-native: main.go
	go run $<