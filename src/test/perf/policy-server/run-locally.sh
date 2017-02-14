# !/bin/bash

set -e -u
set -o pipefail

export CONFIG=/tmp/policy-server-config.json

export API="api.bosh-lite.com"
export CF_USER=admin
export CF_PASSWORD=admin

CF_HOME=~/.cf cf api "$API" --skip-ssl-validation
CF_HOME=~/.cf cf auth "$CF_USER" "$CF_PASSWORD"

echo '
{
  "api": "api.bosh-lite.com",
  "admin_user": "admin",
  "admin_password": "admin",
  "apps": 1000,
  "create_new_policies": false,
  "expiration": 5,
  "logs": "/tmp/perf-logs.txt",
  "num_cells": 50,
  "policies_per_app": 3,
  "poll_interval": 30,
  "skip_ssl_validation": true
}
' > $CONFIG

cd $GOPATH
go run src/test/perf/policy-server/main.go --config "${CONFIG}" \
