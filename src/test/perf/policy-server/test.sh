# !/bin/bash

set -e -u
set -o pipefail

export API="api.bosh-lite.com"
export CF_USER=admin
export CF_PASSWORD=admin

CF_HOME=~/.cf cf api "$API" --skip-ssl-validation

cd $GOPATH

go run src/test/perf/policy-server/main.go \
	-apps 1000 \
	-numCells 50 \
	-policiesPerApp 3 \
	-pollInterval 30s \
	-cfUser "$CF_USER" \
	-cfPassword "$CF_PASSWORD" \
	-api "$API" \
  -out "/tmp/out.$(date +%s).txt" \
  -expiration 5m \
  -setup=false
	
