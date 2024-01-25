#!/usr/bin/env bash
# script to run hyperfine to compare native / static / wasm / wasimoff
# runs of the travelling_salesman binary
# https://github.com/sharkdp/hyperfine
set -eu

# timestamp for filenames
ts=$(date +%s)

# alias to run hyperfine over parameters n with given output file
params="2,3,4,5,6,7,8,9,10,11,12,13"
hf() { hyperfine --shell=none --warmup 3 --min-runs 15 --parameter-list n "$params" --export-csv "$@"; }

# paths to binaries
tsp_native="./bin/target/release/tsp"
tsp_static="./bin/target/x86_64-unknown-linux-musl/release/tsp"
tsp_wasm="./bin/target/wasm32-wasi/release/tsp.opt.wasm"

# check if all required binaries exist
if ! (set -x; [[ -x $tsp_native ]] && [[ -x $tsp_static ]] && [[ -r $tsp_wasm ]] ); then
  echo "ERR: some binaries are missing! please run 'make all' in travelling_salesman" >&2
  exit 1
fi

# ---- NATIVE: the native binary compiled in release mode for current machine ----
hf tspfine_${ts}_native.csv -n "native({n})" "$tsp_native rand {n}"

# ---- STATIC: binary compile statically with musl libc ----
hf tspfine_${ts}_static_musl.csv -n "static({n})" "$tsp_static rand {n}"

# --- WASM: optimized webassembly binary run with wasmtime ----
hf tspfine_${ts}_wasmtime.csv -n "wasmtime({n})" "/usr/bin/wasmtime $tsp_wasm rand {n}"


# ---- WASIMOFF: run the tsp task through a local wasimoff broker and chromium provider ----
cat <<EOF
!! The wasimoff test expects it to be started locally already because it's
!! complicated to launch automatically ..
!!   broker/       $ go run ./ 2>/dev/null
!!   webprovider/  $ yarn build && yarn preview
!!   any/          $ chromium --headless=new "http://localhost:4173/#autoconnect=yes&workers=max"
EOF
read -rsp 'Press any key to continue ...' -n1 _;

# upload the binary, to be sure
broker="http://localhost:4080/api/broker/v1"
curl -kH "content-type: application/wasm" "$broker/upload?name=$(basename "$tsp_wasm")" --data-binary "@$tsp_wasm"
sleep 1

# run hyperfine against wasimoff
exec=$(printf '{ "bin": "%s", "exec": [{ "args": [ "rand", "{n}" ] }] }' "$(basename "$tsp_wasm")")
hf tspfine_${ts}_wasimoff.csv -n "wasimoff({n})" "curl -kH 'content-type: application/json' '$broker/run' -d '$exec'"
