#!/bin/bash -eu

function log() {
  local message=$1
  echo "$(date): ${message}"
}

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

  log "killing ${job_name} with pid ${pid}"
  kill "${pid}"
}

function stop_process() {
  local pid
  pid=$(cat "${PIDFILE}")
  log "stop called. found pid ${pid} in ${PIDFILE}"
  stop_gracefully "${pid}"
  if [ $? -eq 0 ]; then
    rm "${PIDFILE}"
  fi
}

stop_process_on_port() {
  local port=$1

  log "checking for processes listening on port ${port}"
  set +e
  local pids
  pids=$(lsof -t -s TCP:LISTEN -i TCP:"${port}")
  set -e
  if [ ! -z "${pids}" ]; then
    log "the following processes are listening on port ${port}"
    lsof -s TCP:LISTEN -i TCP:"${port}"
  else
    log "no processes found listening on port ${port}"
  fi

  for pid in ${pids}; do
    stop_gracefully "${pid}"
  done
}

stop_gracefully() {
  local pid=$1
  log "stopping process with pid ${pid}"
  kill -TERM "${pid}"
  if wait_pid "${pid}" 80 ; then
    return 0
  fi

  log "unable to stop process using SIGTERM after 8 seconds, will now attempt to SIGQUIT"
  kill -QUIT "${pid}" || true
  if wait_pid "${pid}" 20 ; then
    return 0
  fi

  log "unable to stop process using SIGQUIT after 2 seconds, will now attempt to SIGKILL"
  kill -KILL "${pid}" || true
  if wait_pid "${pid}" 20 ; then
    return 0
  fi

  log "unable to stop process using SIGKILL after 2 seconds"
  return 1
}

function write_pid() {
  local healthy=$1
  local pid=$2

  if [ "${healthy}" -eq 0 ]; then
    echo "${pid}" > "${PIDFILE}"
    log "start succeeded. wrote pid ${pid} to ${PIDFILE}"
  else
    kill -9 "${pid}"
    exit 1
  fi
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
