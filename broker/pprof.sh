#!/usr/bin/env bash
# shorthand to fetch+view a pprof profile, e.g.:
# $ ./pprof.sh http://localhost:4080/debug/pprof/allocs
# $ ./pprof.sh http://localhost:4080/debug/pprof/profile?seconds=30

PPROF_TMPDIR=/tmp/wasimoff-pprof \
  go tool pprof -http :8080 -no_browser "$@"
