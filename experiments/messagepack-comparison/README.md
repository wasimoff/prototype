# MessagePack

Testing MessagePack as an alternative to Protobufs. Notes are mostly in `network_exploration.md`.

### Usage

* `go run ./` to encode an example message to JSON, MessagePack and Protobuf.
* `yarn install && node decoder.mjs` to decode the emitted Base64 strings in JS

### Links

* [MessagePack Homepage](https://msgpack.org/)
* [Javascript implementation](https://github.com/msgpack/msgpack-javascript)
* [Go Tinylib implementation](https://github.com/tinylib/msgp) (codegen, really tiny)
* [Go Uptrace implementation](https://msgpack.uptrace.dev/guide/#quickstart) (works with struct tags, needs tweak for compact format)
* [Protobuf-ES](https://buf.build/blog/protobuf-es-the-protocol-buffers-typescript-javascript-runtime-we-all-deserve) is a new TS/JS [runtime](https://github.com/bufbuild/protobuf-es), if I do want to try Protobuf again – it uses code generation too, though
