#!/usr/bin/env bash
# simple http client to interact with the broker
set -eu

# For Protobuf users: see ../broker/messages.proto and use OffloadWasiJobRequest.
# The gist of the expeted JSON is:
# {
#   "parent": { // common parameters
#     "binary": {
#       "ref": "filename.wasm" // or:
#       "blob": "<base64-encoded bytes"
#     },
#     "envs": [ "ENV=var", ... ],
#   },
#   "tasks": [{
#     "args": [ "arg0", "rand", "10" ],
#     "rootfs": { /* ZIP file, like binary above */ },
#     "artifacts": [ "/hello.txt", ... ] // sent back as ZIP
#   }, { ... }]
# }

# set the URL to the broker here
BROKER="${BROKER:-http://localhost:4080}/api/broker/v1"

# run a json configuration
runjson() { # $1: run configuration
  # you can convert your old configs with:
  # $ jq '. as $t | { parent: { binary: { ref: $t.bin } }, tasks: $t.exec | map({ args: ([$t.bin] + .args), stdin: .stdin | @base64 }) }' config.json
  # upload run configuration and show the result 
  curl --fail-with-body -kX POST -H "content-type: application/json" "$BROKER/run" --data-binary "@$1"
}

# create a run config from arguments
execute() { # $@ arguments
  bin="${1:?first argument is the binary}";
  # create a config with jq
  config=$(jq -cn --args --arg bin "$bin" '{ tasks: [{ binary: { ref: $bin }, args: $ARGS.positional }] }' "$@")
  echo "$config" >&2
  # and run ad-hoc json
  runjson <(echo "$config")
}

# parse the response with jq and decode stdout and stderr to text; use in a pipe
parseresponse() {
  slurp=$(cat)
  jq -r '.tasks[].result | { status, stdout: .stdout | @base64d, stderr: .stderr | @base64d }' <<<"$slurp"
  if [[ $? -ne 0 ]]; then
    echo "failed parsing response; printing raw instead:"
    echo "$slurp"
  fi
}


# upload a file to the broker and receive the ref for your jsons
upload() { # $1: the file to upload, $2: the filename
  # use either explicit filename or basename of file
  name="${2:-$(basename "$1")}"
  # get the mime-type
  mime=$(file -bL --mime-type "$1")
  # upload the file, giving name in query parameter
  curl --fail-with-body -kX POST -H "content-type: $mime" "$BROKER/upload?name=$name" --data-binary "@$1"
}


# first argument is the command
case "${1:-}" in

  upload) upload "${2:?filename required}" "${3:-}" ;;
  run) runjson "${2:?run configuration required}" ;;
  parse) parseresponse ;;
  exec) shift 1; execute "$@" ;;

  *)
    echo >&2 "ERR: unknown command! { run <json>, exec <args>, parse, upload <file> [<name>] }"
    exit 1
  ;;

esac
