#!/bin/bash -eu

function wait_for_server_to_become_healthy() {
  local url=$1
  local timeout=$2
  for _ in $(seq "${timeout}"); do
    set +e
    curl -f --connect-timeout 1 "${url}" > /dev/null 2>&1
    last_exit_code=$?
    if [ $last_exit_code -eq 0  ] || [ $last_exit_code -eq 22  ]; then
      echo 0
      return
    fi
    set -e
    sleep 1
  done

  echo 1
}
