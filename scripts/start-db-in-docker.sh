#!/bin/bash

set -e -u

SCRIPT_PATH="$(cd "$(dirname "${0}")" && pwd)"
. "${SCRIPT_PATH}/start-db-helper"

cd /cf-networking-release
export GOPATH=$PWD


loadIFB
bootDB "${DB:-"notset"}"
set +e
exec /bin/bash
