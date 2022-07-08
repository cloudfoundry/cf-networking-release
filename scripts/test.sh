#!/bin/bash

specificied_package="${1}"

set -e -u

SCRIPT_PATH="$(cd "$(dirname "${0}")" && pwd)"
. "${SCRIPT_PATH}/start-db-helper"

cd "${SCRIPT_PATH}/.."

DB="${DB:-"notset"}"

serial_nodes=1
if [[ "${DB}" == "postgres" ]]; then
  serial_nodes=4
fi

declare -a serial_packages=(
  "src/code.cloudfoundry.org/policy-server/integration/timeouts"
  "src/code.cloudfoundry.org/policy-server/integration"
  "src/code.cloudfoundry.org/policy-server/store/migrations"
  "src/code.cloudfoundry.org/policy-server/store"
)

# smoke/perf/acceptance/scaling tests should be skipped
declare -a ignored_packages=(
  "src/code.cloudfoundry.org/test"
)

for pkg in $(echo "${exclude_packages:-""}" | jq -r .[]); do
  ignored_packages+=("${pkg}")
done

install_ginkgo() {
  if ! [ $(type -P "ginkgo") ]; then
    go install -mod=mod github.com/onsi/ginkgo/ginkgo@v1
  fi
}

containsElement() {
  local e match="$1"
  shift
  for e; do [[ "$e" == "$match" ]] && return 0; done
  return 1
}

test_package() {
  local package=$1
  if [ -z "${package}" ]; then
    return 0
  fi
  shift
  pushd "${package}" &>/dev/null
  ginkgo --race -randomizeAllSpecs -randomizeSuites -failFast \
      -ldflags="extldflags=-WL,--allow-multiple-definition" \
       "${@}";
  rc=$?
  popd &>/dev/null
  return "${rc}"
}

install_ginkgo
bootDB "${DB}"

declare -a packages
if [[ -n "${include_only:-""}" ]]; then
  mapfile -t packages < <(jq -r .[]) <<< "${include_only}"
else
  mapfile -t packages < <(find src -type f -name '*_test.go' -print0 | xargs -0 -L1 -I{} dirname {} | sort -u)
fi

# filter out serial_packages from packages
for i in "${serial_packages[@]}"; do
  packages=("${packages[@]//*$i*}")
done

# filter out explicitly ignored packages
for i in "${ignored_packages[@]}"; do
  packages=("${packages[@]//*$i*}")
  serial_packages=("${serial_packages[@]//*$i*}")
done

if [[ -z "${specificied_package}" ]]; then
  echo "testing packages: " "${packages[@]}"
  for dir in "${packages[@]}"; do
    test_package "${dir}" -p
  done
  echo "testing serial packages: " "${serial_packages[@]}"
  for dir in "${serial_packages[@]}"; do
    test_package "${dir}" --nodes "${serial_nodes}"
  done
else
  specificied_package="${specificied_package#./}"
  if containsElement "${specificied_package}" "${serial_packages[@]}"; then
    echo "testing serial package ${specificied_package}"
    test_package "${specificied_package}" --nodes "${serial_nodes}"
  else
    echo "testing package ${specificied_package}"
    test_package "${specificied_package}" -p
  fi
fi
