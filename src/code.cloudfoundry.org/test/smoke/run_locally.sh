#!/bin/bash

set -e -u

THIS_DIR=$(cd $(dirname $0) && pwd)
cd $THIS_DIR

export CONFIG=/tmp/test-config.json
export APPS_DIR=../../example-apps

echo '
{
  "api": "api.bosh-lite.com",
  "smoke_user": "admin",
  "smoke_password": "admin",
  "app_instances": 2,
  "apps_domain": "bosh-lite.com",
  "prefix":"smoke-",
  "smoke_org": "c2c-smoke"
}
' > $CONFIG

ginkgo -v .
