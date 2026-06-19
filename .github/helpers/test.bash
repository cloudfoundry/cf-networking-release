#!/bin/bash

set -eu
set -o pipefail
GH_WORKSPACE=$1
BUILD_IMAGE=$2
docker run --cap-add=SYS_ADMIN --rm --privileged \
	-v "$GH_WORKSPACE:/workspace" \
	-w /workspace \
	-e MAPPING="$MAPPING" -e DB="$DB" -e DIR="$DIR" -e RUN_AS="$RUN_AS" -e VERIFICATIONS="$VERIFICATIONS" -e FUNCTIONS="$FUNCTIONS" \
	"$BUILD_IMAGE" \
	bash ./ci/shared/tasks/run-bin-test/task.bash --keep-going --trace -r --fail-on-pending --randomize-all --nodes=7 --race --timeout 30m --flake-attempts 2
