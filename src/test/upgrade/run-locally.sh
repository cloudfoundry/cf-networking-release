#!/bin/bash

set -e -u

THIS_DIR=$(cd $(dirname $0) && pwd)
cd $THIS_DIR

export CONFIG=/tmp/test-config.json
export APPS_DIR=../../example-apps
export BASE_MANIFEST=../../../bosh-lite/deployments/diego.yml
export UPGRADE_MANIFEST=../../../bosh-lite/deployments/diego_with_netman.yml

echo '
{
  "api": "api.bosh-lite.com",
  "admin_user": "admin",
  "admin_password": "admin",
  "apps_domain": "bosh-lite.com",
  "skip_ssl_validation": true,
  "use_http": true,
  "bosh_director_url":"https://192.168.50.4:25555",
  "bosh_admin_user":"admin",
  "bosh_admin_password":"admin",
  "bosh_diego_deployment_name":"cf-warden-diego",
  "bosh_cf_deployment_name":"cf-warden",
  "bosh_director_ca_cert":"../../../../bosh-lite/ca/certs/ca.crt"
}
' > $CONFIG

../../../scripts/generate-bosh-lite-manifests

ginkgo -v .
