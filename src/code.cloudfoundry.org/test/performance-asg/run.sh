#!/bin/bash

set -e -u

THIS_DIR=$(cd $(dirname $0) && pwd)
cd $THIS_DIR

export CONFIG=/tmp/test-config.json
export APPS_DIR=../../../example-apps

# Total rules = 
#    global_asgs * asg_size * total_spaces * apps_per_space + 
#    spaces_with_one_asg * asg_size * 1 * apps_per_space +
#    (total_spaces - spaces_with_one_asg) * asg_size * how_many_asgs_is_many * apps_per_space
echo "
{
  \"api\": \"api.sys.pacificblue.cf-app.com\",
  \"admin_user\": \"admin\",
  \"admin_password\": \"${ADMIN_PASSWORD}\",
  \"skip_ssl_validation\": true,
  \"use_http\": true,
  \"concurrency\": 10,
  \"prefix\":\"scale-asg\",
  \"asg_size\": 100,
  \"global_asgs\": 5,
  \"total_spaces\": 2,
  \"spaces_with_one_asg\": 1,
  \"how_many_asgs_is_many\": 2,
  \"apps_per_space\": 1
}
" > $CONFIG

rules=$(jq ' .global_asgs * .asg_size * .total_spaces * .apps_per_space + .spaces_with_one_asg * .asg_size * .apps_per_space + (.total_spaces - .spaces_with_one_asg) * .asg_size * .how_many_asgs_is_many * .apps_per_space' < $CONFIG)
cells=$(bosh vms | grep -iE 'compute|cell' | wc -l)
rules_per_cell=$(expr $rules / $cells)
echo "Targeting ~$rules_per_cell rules per cell ($rules total rules / $cells cells)."
echo "Sleeping for 10s to let you cancel before getting started..."
sleep 10
go run ../../cf-pusher/cmd/multispace-pusher/main.go --config "${CONFIG}"


# Cleanup scripts
# cf delete-org scale-asg-org -f
# cf security-groups | grep 'scale-asg-' | cut -d' ' -f1 | xargs -n1 -P16 cf delete-security-group -f
