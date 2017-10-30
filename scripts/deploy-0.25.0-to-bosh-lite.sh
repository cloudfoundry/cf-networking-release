#!/bin/bash
set -xeu


WORKING_DIR=$(mktemp -d)

CF_NETWORKING_VERSION='0.25.1'
CF_NETWORKING_DEPLOY_VERSION='0.25.0'

pushd "${WORKING_DIR}"
  git clone https://github.com/cloudfoundry/cf-deployment.git
  git clone https://github.com/cloudfoundry/cf-networking-release.git
  git clone git@github.com:cloudfoundry/cf-networking-deployments.git
  git clone git@github.com:cf-container-networking/c2c-workspace.git

  pushd cf-deployment
    CF_DEPLOYMENT_COMMIT_SHA=$(git log -G "url: https:\/\/bosh.io\/d\/github.com\/cloudfoundry\-incubator\/cf-networking-release\?v=$CF_NETWORKING_DEPLOY_VERSION" --format=%H --reverse | head -n1)
    git checkout "${CF_DEPLOYMENT_COMMIT_SHA}"
    bosh -n ucc ./bosh-lite/cloud-config.yml
  popd

  pushd cf-networking-release
    # TAG_DATE=$(git show v${CF_NETWORKING_VERSION} --format=%ai | awk '{ print $1 }' | head -n1)
    TAG_DATE=$(git log -1 v${CF_NETWORKING_DEPLOY_VERSION} --format=%ai | awk '{ print $1 }')
    git checkout "v${CF_NETWORKING_VERSION}"
    ./scripts/update
    bosh create-release --force --timestamp-version && bosh upload-release
  popd

  pushd cf-networking-deployments
    git co "$(git log --since="${TAG_DATE}" --reverse  --format=%H | head -n1)"
  popd

  pushd c2c-workspace
    git checkout "$(git log --since="${TAG_DATE}" --reverse  --format=%H | head -n1)"
    sed -i '' "s#~/workspace#${WORKING_DIR}#" shared.bash
    sed -i '' 's/^main/#main/' shared.bash
    source ./shared.bash
  popd
popd

deploy_bosh_lite
