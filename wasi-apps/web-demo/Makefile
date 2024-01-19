.PHONY: serve
serve: exe.wasm
	python -m http.server

link-tinygo:
link-rust:
link-wat:
link-%:
	cd ../helloworld/$* && make exe.wasm
	ln -sf ../helloworld/$*/exe.wasm
