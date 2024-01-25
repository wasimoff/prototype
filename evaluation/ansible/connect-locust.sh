#!/usr/bin/env bash
# connect to the locust host with local port forwarding for the web frontend

# get the host and username from inventory
address=$(ansible-inventory --list -l "locust[0]" |\
  jq -r '._meta.hostvars | to_entries[] | [ .value.ansible_user, .key ] | join("@")')

# start ssh shell
ssh -t \
  -L "8089:localhost:8089" \
  -o ControlPath=none \
  "$address" #'cd ~/locust && exec bash -il'

# after connection closes, sync all the results
if true; then
  scenario=$(./getivar.sh "{{ scenario }}/")
  results=$(./getivar.sh "{{ pwd_results }}")
  echo -e "\033[1mSyncing results. Expecting: $scenario\033[0m"
  rsync --archive --progress locust.ansemjo.de:results/ "$results/"
fi