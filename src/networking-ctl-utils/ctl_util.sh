#!/bin/bash -eu

function wait_for_server_to_become_healthy() {
  local url=$1
  local timeout=$2
  for _ in $(seq "${timeout}"); do
    set +e
    curl -f --connect-timeout 1 "${url}" > /dev/null 2>&1
    if [ $? -eq 0 ]; then
      echo 0
      return
    fi
    set -e
    sleep 1
  done

  echo 1
}
