#!/bin/bash

set -eu
set -o pipefail

THIS_FILE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
CI="${THIS_FILE_DIR}/../../wg-app-platform-runtime-ci"
. "$CI/shared/helpers/release-note-helpers.bash"
. "$CI/shared/helpers/git-helpers.bash"
REPO_NAME=$(git_get_remote_name)
REPO_PATH="${THIS_FILE_DIR}/../"
unset THIS_FILE_DIR


START_REF="${1}";
END_REF="${2}";
GO_MOD_LOCATION="src/code.cloudfoundry.org/go.mod";

get_non_bot_commits "${START_REF}" "${END_REF}"
echo ""
display_go_mod_diff "${START_REF}" "${END_REF}" "${GO_MOD_LOCATION}"
