#!/bin/bash

specified_package="${1}"

set -e -u

DB="${DB:-"notset"}"

SCRIPT_PATH="$(cd "$(dirname "${0}")" && pwd)"
. "${SCRIPT_PATH}/start-db-helper"

cd "${SCRIPT_PATH}/.."

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
  "src/code.cloudfoundry.org/test/.*"
)

for pkg in $(echo "${exclude_packages:-""}" | jq -r .[]); do
  ignored_packages+=("${pkg}")
done

contains_element() {
  local e match="${1}"
  shift
  for e; do [[ "${e}" == "${match}" ]] && return 0; done
  return 1
}

test_package() {
  local package=$1
  shift
  pushd "${package}" &>/dev/null
  ginkgo --race -randomizeAllSpecs -randomizeSuites -failFast \
      -ldflags="extldflags=-WL,--allow-multiple-definition" \
       "${@}";
  rc=$?
  popd &>/dev/null
  return "${rc}"
}

loadIFB
bootDB "${DB}"

declare -a packages
if [[ -n "${include_only:-""}" ]]; then
  mapfile -t packages < <(jq -r .[] <<< "${include_only}")
else
  mapfile -t packages < <(find src -type f -name '*_test.go' -print0 | xargs -0 -L 1 -I{} dirname {} | sort -u)
fi

# filter out serial_packages from packages
for serial_pkg in "${serial_packages[@]}"; do
    for i in "${!packages[@]}"; do
      if [[ "${packages["${i}"]}" == "${serial_pkg}" ]]; then
        unset "packages[${i}]"
      fi
    done
done

# filter out explicitly ignored packages
for ignored_pkg in "${ignored_packages[@]}"; do
  for i in "${!packages[@]}"; do
    if [[ "${packages["${i}"]}" =~ ^${ignored_pkg}$ ]]; then
      unset "packages[${i}]"
    fi
  done
  for i in "${!serial_packages[@]}"; do
    if [[ "${serial_packages["${i}"]}" =~ ^${ignored_pkg}$ ]]; then
      unset "serial_packages[${i}]"
    fi
  done
done

if [[ -z "${specified_package}" ]]; then
  echo "testing packages: " "${packages[@]}"
  for dir in "${packages[@]}"; do
    echo testing "${dir}"
    test_package "${dir}" -p
  done
  echo "testing serial packages: " "${serial_packages[@]}"
  for dir in "${serial_packages[@]}"; do
    echo testing "${dir}"
    test_package "${dir}" --nodes "${serial_nodes}"
  done
else
  specified_package="${specified_package#./}"
  if contains_element "${specified_package}" "${serial_packages[@]}"; then
    echo "testing serial package ${specified_package}"
    test_package "${specified_package}" --nodes "${serial_nodes}"
  else
    echo "testing package ${specified_package}"
    test_package "${specified_package}" -p
  fi
fi
