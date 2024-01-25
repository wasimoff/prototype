#!/usr/bin/env bash
# this script calls the create-servers playbook and then logs the
# actually created hcloud setup to the scenario directory
set -eu

# get scenario directory
scenario="$(./getivar.sh "{{ pwd_results }}/{{ scenario }}")"

# warning if it already exists
if [[ -e $scenario/servers.txt ]]; then
  echo -e "\033[1;31mWARNING: this scenario directory exists already!\033[0m" >&2
  echo "--> $scenario" >&2
  read -rsp 'Press any key to continue ...' -n1 _;
  echo;
fi
set -x
mkdir -p "$scenario"

# run the playbook
ansible-playbook ./01-create-servers.yaml

# log created hcloud setup
hcloud server list -o columns=id,name,type,location,ipv4,ipv6 > "$scenario/servers.txt"