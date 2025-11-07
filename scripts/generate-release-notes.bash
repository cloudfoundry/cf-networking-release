#!/bin/bash

set -eu
set -o pipefail

THIS_FILE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
CI="${THIS_FILE_DIR}/../../wg-app-platform-runtime-ci"
. "$CI/shared/helpers/release-note-helpers.bash"
. "$CI/shared/helpers/git-helpers.bash"
unset THIS_FILE_DIR

# ex. version_range="v0.343.0...v0.344.0"
version_range="${1:?Please provide the start and end versions you want to generate release notes for './generate-release-notes.bash start_ref...end_ref' }"
# ex. local_start_ref="v0.343.0"
local_start_ref=$(get_start_ref_from_range "${version_range}")
# ex. local_end_ref="v0.344.0"
local_end_ref=$(get_end_ref_from_range "${version_range}")

GO_MOD_LOCATION="src/code.cloudfoundry.org/go.mod";

get_non_bot_commits "${local_start_ref}" "${local_end_ref}"
display_go_mod_diff "${local_start_ref}" "${local_end_ref}" "${GO_MOD_LOCATION}"
