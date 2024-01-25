#!/usr/bin/env bash
# remote start line and rename labels + append colours
set -eu

sed \
  -e 's/step_ms;label/&;colour/' \
  -e '/client: start/d' \
  -e 's/broker: request received/transmit request;1/' \
  -e 's/broker: configuration decoded/decode request;2/' \
  -e 's/broker: task queued/queue task;2/' \
  -e 's/broker: task scheduled/schedule task;2/' \
  -e 's/provider: rpc: function top/transmit RPC;2/' \
  -e '/provider: rpc: parsed options/d' \
  -e 's/provider: rpc: pool.exec got a worker/get a Worker;4/' \
  -e 's/provider: worker: function top/transmit task;4/' \
  -e 's/provider: worker: commandline logged/log arguments;4/' \
  -e 's/provider: worker: filesystem prepared/prepare filesystem;4/' \
  -e 's/provider: worker: wasm module compiled/get cached module;4/' \
  -e 's/provider: worker: wasi shim prepared/prepare WASI shim;4/' \
  -e 's/provider: worker: module instantiated/instantiate module;4/' \
  -e 's/provider: worker: task completed/execute task;4/' \
  -e 's/broker: task rpc completed/respond to RPC;2/' \
  -e 's/client: response received/respond to client;1/' \
  "$@"