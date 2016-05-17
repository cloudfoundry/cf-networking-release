#!/bin/bash

set -e -u

cd netman-release
export GOPATH=$PWD

declare -a packages=(
  "src/github.com/cloudfoundry-incubator/guardian-cni-adapter"
  )

for dir in "${packages[@]}"; do
  pushd $dir
    extraFlags="${GINKGO_EXTRA_FLAGS:-""}"
    ginkgo -r -p -randomizeAllSpecs -randomizeSuites $extraFlags "$@"
  popd
done
