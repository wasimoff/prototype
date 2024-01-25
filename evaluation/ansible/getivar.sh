#!/usr/bin/env bash
# get a message templated from inventory
ansible -m debug -a msg="$1" localhost | sed '1s/^.* => //' | jq -r .msg