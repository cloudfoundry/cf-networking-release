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
NUM_SAMPLES=15

rm -f $STATS_FILE

function purgeAllPolicies() {
  echo -n "purging all policies"
  cf curl /networking/v0/external/policies/delete -d @<(cf curl /networking/v0/external/policies)
  echo "done"
}

function runTest() {
  NUM_EXISTING_POLICIES=$1
  OAUTH_TOKEN="$(cf oauth-token)"  # refresh this frequently, since it expires
  go run \
    src/netman-cf-perf/latency-to-enforce-with-many-policies/main.go \
    "$OAUTH_TOKEN" \
    $NUM_EXISTING_POLICIES \
    $BASE_URL \
    $PROXY_APP_GUID \
    $PROXY_APP_ROUTE \
    $STATS_FILE \
    $NUM_SAMPLES \
    $BACKEND_APP_GUID \
    $BACKEND_APP_ROUTE
}

purgeAllPolicies

runTest 10
runTest 100
runTest 500
runTest 1000
runTest 2000
runTest 4000
runTest 8000

purgeAllPolicies
