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
  "admin_secret": "{{admin-secret}}",
  "apps_domain": "bosh-lite.com",
  "skip_ssl_validation": true,
  "test_app_instances": 2,
  "test_applications": 2,
  "proxy_instances": 1,
  "proxy_applications": 1,
  "extra_listen_ports": 2,
  "prefix":"cf-networking-test-"
}
' > ${CONFIG}

ADMIN_PASSWORD=`grep cf_admin_password ${VARS_STORE} | cut -d' ' -f2`
sed -i -- "s/{{admin-password}}/${ADMIN_PASSWORD}/g" /tmp/test-config.json
ADMIN_SECRET=`grep uaa_admin_client_secret ${VARS_STORE} | cut -d' ' -f2`
sed -i -- "s/{{admin-secret}}/${ADMIN_SECRET}/g" /tmp/test-config.json

ginkgo -v .
