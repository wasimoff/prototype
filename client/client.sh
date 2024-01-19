#!/usr/bin/env bash
# simple http client to interact with the broker
set -eu

# CURL vs. HTTPIE
# this script shows curl [https://curl.se/] and httpie [https://httpie.io/] commands
# side-by-side. use the one that you prefer. curl is faster but httpie formats
# the output nicer ...

# set the URL to the broker here
BROKER="${BROKER:-http://localhost:4080}/api/broker/v1"

# run a json configuration
runjson() { # $1: run configuration

  # upload run configuration and show the result 
  #curl -kX POST -H "content-type: application/json" "$BROKER/run" --data "@$1"
  http --verify=false "$BROKER/run" "@$1"

}

# create a run config from arguments
execute() { # $@ arguments
  #! THIS IS VERY EXPERIMENTAL
  bin="${1:?first argument is the binary}"; shift 1;
  args="$([[ -n ${1+defined} ]] && printf '"%s",' "$@" || printf ".")"
  config=$(printf '{ "bin":"%s", "exec":[{ "args": [%s]}] }' "$bin" "${args::-1}")
  #curl -kX POST -H "content-type: application/json" "$BROKER/run" <<<"$config"
  http -v --verify=false "$BROKER/run" content-type:application/json <<<"$config"
}

# upload a file to the providers in preparation
upload() { # $1: the file to upload, $2: the filename

  # use either explicit filename or basename of file
  name="${2:-$(basename "$1")}"

  # upload the file, giving name in query parameter
  #curl -kX POST -H "content-type: application/wasm" "$BROKER/upload?name=$name" --data-binary "@$1"
  http --verify=false --ignore-stdin "$BROKER/upload" "@$1" name=="$name"

}

# first argument is the command
case "${1:-}" in

  upload) upload "${2:?filename required}" "${3:-}" ;;
  run) runjson "${2:?run configuration required}" ;;
  exec) shift 1; execute "$@" ;;

  *)
    echo >&2 "ERR: unknown command! { run <json>, upload <file> [<name>] }"
    exit 1
  ;;

esac
