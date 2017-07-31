#!/bin/bash

set -e -u

THIS_DIR=$(cd $(dirname $0) && pwd)
cd $THIS_DIR

export CONFIG=/tmp/test-config.json
export APPS_DIR=../../example-apps

VARS_STORE="$HOME/workspace/cf-networking-deployments/environments/local/deployment-vars.yml"

echo '
{
  "api": "api.bosh-lite.com",
  "admin_user": "admin",
  "admin_password": "{{admin-password}}",
  "apps_domain": "bosh-lite.com",
  "skip_ssl_validation": true,
  "use_http": true,
  "test_app_instances": 2,
  "test_applications": 4,
	"test_app_registry_ttl_seconds": 15,
  "policy_update_wait_seconds": 10,
  "proxy_applications": 3,
  "proxy_instances": 2,
  "sample_percent":20,
  "concurrency": 2,
  "prefix":"scale-"
}
' > $CONFIG

ADMIN_PASSWORD=`grep cf_admin_password ${VARS_STORE} | cut -d' ' -f2`
sed -i -- "s/{{admin-password}}/${ADMIN_PASSWORD}/g" /tmp/test-config.json

go run ../../cf-pusher/cmd/cf-pusher/main.go --config "${CONFIG}"
ginkgo -v .
