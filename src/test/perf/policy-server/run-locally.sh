# !/bin/bash

set -e -u
set -o pipefail

go build -o /tmp/perf-tester main.go

export CONFIG=/tmp/policy-server-config.json

export API="api.bosh-lite.com"
export CF_USER=admin
export CF_PASSWORD=admin

# From inside the cluster, use this:
#export POLICY_SERVER_INTERNAL_BASE_URL=https://policy-server.service.cf.internal:4003
#export POLICY_SERVER_CERTS_DIR="/var/vcap/jobs/vxlan-policy-agent/config/certs"

# for testing on bosh-lite from outside the cluster:
export POLICY_SERVER_INTERNAL_BASE_URL="https://10.244.16.2:4003"
export POLICY_SERVER_CERTS_DIR="/tmp/policy-server-certs"

cf api "$API" --skip-ssl-validation
cf auth "$CF_USER" "$CF_PASSWORD"

echo "
{
  \"api\": \"${API}\",
  \"apps\": 2,
  \"create_new_policies\": false,
  \"test_duration_minutes\": 2,
  \"logs\": \"/tmp/perf-logs.txt\",
  \"num_cells\": 2,
  \"policies_per_app\": 3,
  \"poll_interval\": 5,

  \"policy_server_internal_base_url\": \"${POLICY_SERVER_INTERNAL_BASE_URL}\",
  \"ca_cert_file\": \"${POLICY_SERVER_CERTS_DIR}/ca.crt\",
  \"client_cert_file\": \"${POLICY_SERVER_CERTS_DIR}/client.crt\",
  \"client_key_file\": \"${POLICY_SERVER_CERTS_DIR}/client.key\"
}
" > $CONFIG

/tmp/perf-tester --config "${CONFIG}" \
