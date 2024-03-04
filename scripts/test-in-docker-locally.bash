#!/bin/bash

set -eu

THIS_FILE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
. "${THIS_FILE_DIR}/local-testing-helper.sh"

test_in_docker_locally "$@"
