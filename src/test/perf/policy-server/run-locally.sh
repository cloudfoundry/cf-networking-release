# !/bin/bash

set -e -u
set -o pipefail

export CONFIG=/tmp/policy-server-config.json

export API="api.bosh-lite.com"
export CF_USER=admin
export CF_PASSWORD=admin

CF_HOME=~/.cf cf api "$API" --skip-ssl-validation
CF_HOME=~/.cf cf auth "$CF_USER" "$CF_PASSWORD"

echo "
{
  \"api\": \"${API}\",
  \"admin_user\": \"${CF_USER}\",
  \"admin_password\": \"${CF_PASSWORD}\",
  \"apps\": 10000,
  \"create_new_policies\": false,
  \"expiration\": 50,
  \"logs\": \"/tmp/perf-logs.txt\",
  \"num_cells\": 100,
  \"policies_per_app\": 3,
  \"poll_interval\": 5,
  \"skip_ssl_validation\": true
}
" > $CONFIG

cd $GOPATH
go run src/test/perf/policy-server/main.go --config "${CONFIG}" \
