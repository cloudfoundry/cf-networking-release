#!/bin/bash

set -e -u
set -o pipefail

# set -x  # uncomment this to debug this script

cd $GOPATH

# pushd src/example-apps/proxy
#   cf api api.bosh-lite.com --skip-ssl-validation
#   cf auth admin admin
#   cf target -o o -s s
#   cf push proxy
#   cf push proxy-backend
# popd

PROXY_APP_NAME=proxy
BACKEND_APP_NAME=proxy-backend
PROXY_APP_GUID=$(cf app --guid $PROXY_APP_NAME)
BACKEND_APP_GUID=$(cf app --guid $BACKEND_APP_NAME)
BASE_URL=https://api.bosh-lite.com
PROXY_APP_ROUTE=http://proxy.bosh-lite.com
BACKEND_APP_ROUTE=http://proxy-backend.bosh-lite.com
STATS_FILE=/tmp/latency_stats
NUM_SAMPLES=7

rm -f $STATS_FILE

function purgeAllPolicies() {
  echo -n "purging all policies"
  cf curl /networking/v0/external/policies/delete -d @<(cf curl /networking/v0/external/policies)
  echo "done"
}

function runTest() {
  pushd src/test/perf/latency-to-enforce-with-many-asgs/
    NUM_EXISTING_ASGS=$1
    OAUTH_TOKEN="$(cf oauth-token)"  # refresh this frequently, since it expires

    go run build_big_asg/build_big_asg.go $NUM_EXISTING_ASGS > /tmp/big-asg.go
    cf create-security-group big-asg /tmp/big-asg.go
    cf bind-security-group big-asg o s

    cf restart proxy &
    cf restart proxy-backend &
    wait

    go run \
      get_samples/get_samples.go \
      "$OAUTH_TOKEN" \
      $NUM_EXISTING_ASGS \
      $BASE_URL \
      $PROXY_APP_GUID \
      $PROXY_APP_ROUTE \
      $STATS_FILE \
      $NUM_SAMPLES \
      $BACKEND_APP_GUID \
      $BACKEND_APP_ROUTE

    cf unbind-security-group big-asg o s
    cf delete-security-group big-asg -f
  popd
}

purgeAllPolicies

runTest 1000
runTest 2000
runTest 5000
runTest 10000
runTest 50000

purgeAllPolicies
