exe.wasm: hello.wat
	wat2wasm -o $@ $<

.PHONY: run
run: exe.wasm
	wasmtime run --dir ./ -- $< argument
