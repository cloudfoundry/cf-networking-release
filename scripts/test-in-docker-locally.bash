#!/bin/bash

set -eu

THIS_FILE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
CI="${THIS_FILE_DIR}/../../wg-app-platform-runtime-ci"
. "$CI/shared/helpers/git-helpers.bash"
REPO_NAME=$(git_get_remote_name)

if [[ ${DB:-empty} == "empty" ]]; then
  DB=mysql
fi

DB="${DB}" "${THIS_FILE_DIR}/create-docker-container.bash" '/repo/scripts/docker/tests-templates.bash'
DB="${DB}" "${THIS_FILE_DIR}/create-docker-container.bash" '/repo/scripts/docker/test.bash' "$@"
DB="${DB}" "${THIS_FILE_DIR}/create-docker-container.bash" '/repo/scripts/docker/lint.bash'
