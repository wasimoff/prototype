#!/usr/bin/env bash
# connect to the wasimoff providers and start chromium-shells

# collect an array with the providers
providers=($(ansible-inventory --list -l "provider" |\
  jq -r '._meta.hostvars | to_entries[] | [ .value.ansible_user, .key ] | join("@")'))

# possibly pass a 'show' parameter to connect.sh here to see console
args=""

# this tmux command assumes that there are at least two providers
tmuxscript=("new-session" "-s" "providers")
tmuxscript+=("-n" "${providers[0]}" "ssh -t ${providers[0]} './connect.sh $args'" \;)
for p in "${providers[@]:1}"; do
  tmuxscript+=("new-window" "-n" "$p" "ssh -t $p './connect.sh $args'" \;)
done
#tmuxscript+=("select-layout" "even-vertical")

tmux "${tmuxscript[@]}"
