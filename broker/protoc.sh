#!/usr/bin/env bash
# regenerate protobuf codegen here and in webprovider
self=${BASH_SOURCE[0]}

if [[ $1 = -w ]]; then

  # run once, then continuously watch with inotify and rerun
  ./protoc.sh
  inotifywait -m -e close_write messages.proto | while read; do ./protoc.sh; done

else

  # or run only once
  set -ex
  (cd net/pb/ && go generate)
  (cd ../webprovider/ && yarn protoc)

fi
