#!/bin/bash

set -e -u -x

SCRIPT_PATH="$(cd "$(dirname $0)" && pwd)"

cd ${SCRIPT_PATH}/..

SERIAL_NODES="${SERIAL_NODES:-1}"

declare -a serial_packages=(
  "src/code.cloudfoundry.org/policy-server/integration/timeouts"
  "src/code.cloudfoundry.org/policy-server/integration"
  "src/code.cloudfoundry.org/policy-server/store/migrations"
  "src/code.cloudfoundry.org/policy-server/store"
  "src/example-apps/tick"
)

# smoke/perf/acceptance/scaling tests should be skipped
declare -a ignored_packages=(
  "src/code.cloudfoundry.org/test"
)

for pkg in $(echo "${exclude_packages:-""}" | jq -r .[]); do
  ignored_packages+=("${pkg}")
done

function loadIFB {
  set +e
    depmod $(uname -r)
    modprobe ifb
  set -e
}

function bootDB {
  db=$1

  if [ "$db" = "random" ]; then
    cointoss=$RANDOM
    set +e
    let "cointoss %= 2"
    set -e
    if [ "$cointoss" == "0" ]; then
      db="postgres"
    else
      db="mysql"
    fi
  fi

  if [ "$db" = "postgres" ]; then
    launchDB="(docker-entrypoint.sh postgres &> /var/log/postgres-boot.log) &"
    testConnection="psql -h localhost -U postgres -c '\conninfo' &>/dev/null"
  elif [ "$db" = "mysql" ]; then
    chown -R mysql:mysql /var/run/mysqld
    launchDB="(MYSQL_ROOT_PASSWORD=password /entrypoint.sh mysqld &> /var/log/mysql-boot.log) &"
    testConnection="echo '\s;' | mysql -h 127.0.0.1 -u root --password='password' &>/dev/null"
  else
    echo "skipping database"
    return 0
  fi

  echo -n "booting $db"
  eval "$launchDB"
  for _ in $(seq 1 60); do
    set +e
    eval "${testConnection}"
    exitcode=$?
    set -e
    if [ ${exitcode} -eq 0 ]; then
      echo "connection established to $db"
      return 0
    fi
    echo -n "."
    sleep 1
  done
  echo "unable to connect to $db"
  exit 1
}

loadIFB
bootDB "${DB:-"notset"}"

declare -a packages
if [[ -n "${include_only}" ]]; then
  packages=
  mapfile -t packages < <(jq -r ,[]) <<< "${include_only}"
else
  packages=($(find src -type f -name "*_test.go" | xargs -L 1 -I{} dirname {} | sort -u))
fi

# filter out serial_packages from packages
for i in "${serial_packages[@]}"; do
  packages=(${packages[@]//*$i*})
done

# filter out explicitly ignored packages
for i in "${ignored_packages[@]}"; do
  packages=(${packages[@]//*$i*})
  serial_packages=(${serial_packages[@]//*$i*})
done

if [ "${1:-""}" = "" ]; then
  for dir in "${packages[@]}"; do
    pushd "$dir"
      ginkgo -p --race -randomizeAllSpecs -randomizeSuites \
        -ldflags="-extldflags=-Wl,--allow-multiple-definition" \
        ${@:2}
    popd
  done
  for dir in "${serial_packages[@]}"; do
    pushd "$dir"
      ginkgo --nodes "${SERIAL_NODES}" --race -randomizeAllSpecs -randomizeSuites -failFast \
        -ldflags="-extldflags=-Wl,--allow-multiple-definition" \
        ${@:2}
    popd
  done
else
  dir="${@: -1}"
  dir="${dir#./}"
  for package in "${serial_packages[@]}"; do
    if [[ "${dir##$package}" != "${dir}" ]]; then
      ginkgo --race -randomizeAllSpecs -randomizeSuites -failFast \
        -ldflags="-extldflags=-Wl,--allow-multiple-definition" \
        "${@}"
      exit $?
    fi
  done
  ginkgo -p --race -randomizeAllSpecs -randomizeSuites -failFast \
    -ldflags="-extldflags=-Wl,--allow-multiple-definition" \
    "${@}"
fi
