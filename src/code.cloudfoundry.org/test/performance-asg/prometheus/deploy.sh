#!/usr/bin/env bash

set -eu


BOSH_ENVIRONMENT=${BOSH_ENVIRONMENT:?must set BOSH_ENVIRONMENT env var}
BOSH_CA_CERT=${BOSH_CA_CERT:?must set BOSH_CA_CERT env var}

BOSH_CA_ARG="-v"
if ! echo "${BOSH_CA_CERT}" | grep "END CERTIFICATE" >& /dev/null; then
  BOSH_CA_ARG="--var-file"
fi

PROMETHEUS_BOSH_RELEASE_DIR=${PROMETHEUS_BOSH_RELEASE_DIR:=~/workspace/prometheus-boshrelease}
ASG_TEST_DIR=${ASG_TEST_DIR:=~/workspace/cf-networking-release/src/code.cloudfoundry.org/test/performance-asg/prometheus}

bosh -d prometheus deploy "${PROMETHEUS_BOSH_RELEASE_DIR}/manifests/prometheus.yml" \
  --vars-store "${ASG_TEST_DIR}/change-me-deployment-vars.yml" \
  -o "${PROMETHEUS_BOSH_RELEASE_DIR}/manifests/operators/monitor-bosh.yml" \
  -v bosh_url="${BOSH_ENVIRONMENT}" -v "uaa_bosh_exporter_client_id=${BOSH_CLIENT}" \
  -v "uaa_bosh_exporter_client_secret=${BOSH_CLIENT_SECRET}" \
  "${BOSH_CA_ARG}" "bosh_ca_cert=${BOSH_CA_CERT}" \
  -o "${PROMETHEUS_BOSH_RELEASE_DIR}/manifests/operators/monitor-cf.yml" \
  -o "${PROMETHEUS_BOSH_RELEASE_DIR}/manifests/operators/enable-cf-route-registrar.yml" \
  -o "${PROMETHEUS_BOSH_RELEASE_DIR}/manifests/operators/enable-cf-api-v3.yml" \
  -o "${PROMETHEUS_BOSH_RELEASE_DIR}/manifests/operators/enable-cf-loggregator-v2.yml" \
  -o "${PROMETHEUS_BOSH_RELEASE_DIR}/manifests/operators/enable-bosh-uaa.yml" \
  -o "${ASG_TEST_DIR}/change-prometheus-defaults.yml" \
  "$@"
