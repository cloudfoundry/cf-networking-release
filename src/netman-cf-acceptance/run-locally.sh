#!/bin/bash

set -e -u

THIS_DIR=$(cd $(dirname $0) && pwd)
cd $THIS_DIR

export CONFIG=/tmp/bosh-lite-integration-config.json
export APPS_DIR=../example-apps

echo '
{
  "api": "api.bosh-lite.com",
  "admin_user": "admin",
  "admin_password": "admin",
  "apps_domain": "bosh-lite.com",
  "skip_ssl_validation": true,
  "use_http": true,
  "test_user": "admin",
  "test_user_password": "admin",
  "test_app_instances": 4,
  "test_applications": 1
}
' > $CONFIG

ginkgo -v .
