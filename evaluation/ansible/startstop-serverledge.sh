#!/usr/bin/env bash
# wrapper for the ansible playbook with the same name to pass argument easier

case "$1" in
  restart*) state=restarted ;;
  start*)   state=started ;;
  stop*)    state=stopped ;;
  *) echo "required: { start | stop }" >&2; exit 1 ;;
esac

ansible-playbook ./startstop-serverledge.yaml -e "state=$state"