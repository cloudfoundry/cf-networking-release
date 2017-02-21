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
  "admin_secret": "admin-secret",
  "apps_domain": "bosh-lite.com",
  "skip_ssl_validation": true,
  "test_app_instances": 2,
  "test_applications": 2,
  "proxy_instances": 1,
  "proxy_applications": 1,
  "extra_listen_ports": 2,
  "prefix":"test-"
}
' > $CONFIG

ginkgo -v .
