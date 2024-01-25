#!/usr/bin/env bash
# connect to broker and all provider nodes in tmux

# collect an array with the providers
providers=($(ansible-inventory --list -l "broker,provider" |\
  jq -r '._meta.hostvars | to_entries[] | .key'))
  #jq -r '._meta.hostvars | to_entries[] | [ .value.ansible_user, .key ] | join("@")'))

# this tmux script assumes that there are at least two providers
tmuxscript=("new-session" "-s" "cluster")
tmuxscript+=("-n" "${providers[0]}" "ssh -t ${providers[0]}" \;)
for p in "${providers[@]:1}"; do
  tmuxscript+=("new-window" "-n" "$p" "ssh -t $p" \;)
done
# not applicable for windows
#tmuxscript+=("select-layout" "even-vertical")

tmux "${tmuxscript[@]}"
