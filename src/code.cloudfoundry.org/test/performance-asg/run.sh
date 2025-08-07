#!/bin/bash

set -e -u

THIS_DIR=$(cd $(dirname $0) && pwd)
cd $THIS_DIR

export CONFIG=/tmp/test-config.json
export APPS_DIR=${HOME}/workspace
export APP_FOLDER=cf-nc-app

# Total rules =
#    (global_asgs * asg_size * total_spaces * apps_per_space) +
# 20,000,000
# 150,000,000
#    (asgs_with_multiple_spaces * asg_size * space_count_for_asgs_with_multiple_spaces  * apps_per_space) +
# 60,000,000
# 1,500,00
#    ((total_asgs - global_asgs - asgs_with_multiple_spaces) * asg_size * apps_per_space)
# 4,790,000
# 2,970,000

# 84,79,0000 total rules
# 154470000
# 400k per cell
# limited by 7k app instances -> 200k per cell.


# Size of an asg: asg_size * 64 bytes (roughly 64 bytes per rule, asg_size is number of rules)
#ADMIN_PASSWORD="$(credhub get -n "$(credhub find -n cf_admin_password -j | jq -r .credentials[0].name)" -j | jq -r .value)"
ADMIN_PASSWORD="Evek_edv2bXMy46kxyh_H4R8mikm78aa"
CF_API=https://api.sys-tpcf-dasg.lvn.broadcom.net
export CF_DIAL_TIMEOUT=60
echo "
{
  \"api\": \"${CF_API}\",
  \"admin_user\": \"admin\",
  \"admin_password\": \"${ADMIN_PASSWORD}\",
  \"skip_ssl_validation\": true,
  \"use_http\": true,
  \"concurrency\": 12,
  \"prefix\":\"scale-asg\",
  \"total_asgs\":40000,
  \"total_spaces\": 15000,
  \"asg_size\": 200,
  \"global_asgs\": 50,
  \"asgs_with_multiple_spaces\": 100,
  \"space_count_for_asgs_with_multiple_spaces\": 75,
  \"apps_per_space\": 1,
  \"max_app_instances\": 7000,
  \"app_instances_per_app\": 1,
  \"skip_asg_creation\": true
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
