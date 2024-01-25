#!/usr/bin/env bash
# rerun gnuplot whenever the plotscript is saved

# arguments required, incl. script
args=("$@")
if [[ ${#args} -lt 1 ]]; then
  echo "pass gnuplot arguments (including script in last arg)!" >&2
  exit 1
fi

# plot once to begin
gnuplot "${args[@]}"

# watch script and replot every time you save
while inotifywait -e close_write "${args[-1]}" 2>/dev/null; do
  gnuplot "${args[@]}"
done
