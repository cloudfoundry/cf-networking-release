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

function handle_orphaned_server() {
  local job_name=$1
  local pid=$2
  echo "killing ${job_name} with pid ${pid}"
  kill "${pid}"
}

function wait_pid() {
  local pid=$1
  local max_checks=$2

  checks="${max_checks}"
  while [ -e /proc/"${pid}" ]; do
    checks=$((checks - 1))
    if [ "${checks}" -le 0 ]; then
      return 1
    fi
    sleep 0.1
  done

  return 0
}

function stop_process() {
  local pid
  pid=$(cat "${PIDFILE}")

  echo "stopping..."
  kill -TERM "${pid}"
  if wait_pid "${pid}" 100 ; then
    rm "${PIDFILE}"
    return 0
  fi

  echo "unable to stop process using SIGTERM after 10 seconds, will now attempt to SIGQUIT"
  kill -QUIT "${pid}" || true
  if wait_pid "${pid}" 50 ; then
    rm "${PIDFILE}"
    return 0
  fi

  echo "unable to stop process using SIGQUIT after 5 seconds, will now attempt to SIGKILL"
  kill -KILL "${pid}" || true
  if wait_pid "${pid}" 50 ; then
    rm "${PIDFILE}"
    return 0
  fi

  echo "unable to stop process using SIGKILL after 5 seconds"
  return 1
}

function write_pid() {
  local healthy=$1
  local pid=$2

  if [ "${healthy}" -eq 0 ]; then
    echo "${pid}" > "${PIDFILE}"
  else
    kill -9 "${pid}"
    exit 1
  fi
}
