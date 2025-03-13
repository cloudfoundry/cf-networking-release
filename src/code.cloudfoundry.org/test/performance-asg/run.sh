#!/bin/bash

set -e -u

THIS_DIR=$(cd $(dirname $0) && pwd)
cd $THIS_DIR

export CONFIG=/tmp/test-config.json
export APPS_DIR=../../../example-apps

# Total rules =
#    (global_asgs * asg_size * total_spaces * apps_per_space) +
#    (asgs_with_multiple_spaces * asg_size * space_count_for_asgs_with_multiple_spaces  * apps_per_space) +
#    ((total_asgs - global_asgs - asgs_with_multiple_spaces) * asg_size * apps_per_space)

ADMIN_PASSWORD="$(credhub get -n "$(credhub find -n cf_admin_password -j | jq -r .credentials[0].name)" -j | jq -r .value)"
echo "
{
  \"api\": \"${CF_API}\",
  \"admin_user\": \"admin\",
  \"admin_password\": \"${ADMIN_PASSWORD}\",
  \"skip_ssl_validation\": true,
  \"use_http\": true,
  \"concurrency\": 12,
  \"prefix\":\"scale-asg\",
  \"total_asgs\":25000,
  \"total_spaces\": 240,
  \"asg_size\": 100,
  \"global_asgs\": 50,
  \"asgs_with_multiple_spaces\": 75,
  \"space_count_for_asgs_with_multiple_spaces\": 100,
  \"apps_per_space\": 1
}
" > $CONFIG

rules=$(jq '.global_asgs * .asg_size * .total_spaces * .apps_per_space + .asgs_with_multiple_spaces * .asg_size * .space_count_for_asgs_with_multiple_spaces * .apps_per_space + (.total_asgs - .global_asgs - .asgs_with_multiple_spaces) * .asg_size * .apps_per_space' < $CONFIG)
cells=$(bosh vms | grep -iE 'compute|cell' | wc -l)
rules_per_cell=$(expr $rules / $cells)
echo "Targeting ~$rules_per_cell rules per cell ($rules total rules / $cells cells)."
echo "Sleeping for 10s to let you cancel before getting started..."
sleep 10
go run ../../cf-pusher/cmd/multispace-pusher/main.go --config "${CONFIG}"


# Cleanup scripts
# cf delete-org scale-asg-org -f
# cf security-groups | grep 'scale-asg-' | cut -d' ' -f1 | xargs -n1 -P16 cf delete-security-group -f
