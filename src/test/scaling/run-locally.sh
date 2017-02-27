#!/bin/bash

set -e -u

THIS_DIR=$(cd $(dirname $0) && pwd)
cd $THIS_DIR

export CONFIG=/tmp/test-config.json
export APPS_DIR=../../example-apps

echo '
{
  "api": "api.bosh-lite.com",
  "admin_user": "admin",
  "admin_password": "admin",
  "apps_domain": "bosh-lite.com",
  "skip_ssl_validation": true,
  "use_http": true,
  "test_app_instances": 2,
  "test_applications": 4,
  "policy_update_wait_seconds": 10,
  "proxy_applications": 3,
  "proxy_instances": 2,
  "sample_percent":20,
  "concurrency": 2,
  "prefix":"scale-"
}
' > $CONFIG

go run ../../cf-pusher/cmd/cf-pusher/main.go --config "${CONFIG}"
ginkgo -v .
