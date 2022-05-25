#!/bin/bash

set -e -u

THIS_DIR=$(cd $(dirname $0) && pwd)
cd $THIS_DIR

export CONFIG=/tmp/test-config.json
export APPS_DIR=../../../example-apps

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
  \"rule_count_for_space_specific_asgs\": 500,
}
" > $CONFIG

go run ../../cf-pusher/cmd/multispace-pusher/main.go --config "${CONFIG}"


# Cleanup scripts
# cf delete-org scale-asg-org -f
# cf security-groups | grep 'scale-asg-' | cut -d' ' -f1 | xargs -n1 -P16 cf delete-security-group -f
