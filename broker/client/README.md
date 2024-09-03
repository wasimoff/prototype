# client.go

This is a quick reimplementation of the `client.sh` script in Go, to be portable
across any system with a Go compilation target.

Usage is mostly identical but the output format of tasks is generated with
`protobuf/encoding/prototext`, so you won't be able to parse its output with
`jq`. On the other hand, it handles decoding strings in stdout and stderr for
you already.

### Usage

0. Set the origin URL to your Broker: `export BROKER=http://localhost:4080` (default)

1. Upload your WASI preview 1 binary: `./client upload app.wasm`

2. Test an ad-hoc command: `./client exec app.wasm -myarg 0`

3. Save the displayed JSON, modify and rerun: `./client run tasks.json`
