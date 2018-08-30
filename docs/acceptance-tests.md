# Acceptance Tests

The `cf-networking-release` acceptance tests can be run as follows:

```
export APPS_DIR=$WORKSPACE/cf-networking-release/src/example-apps
export CONFIG=$PWD/cnats-config.json

cat << EOF > $CONFIG
{
  "api": "api.cf.domain",
  "admin_user": "admin",
  "admin_password": "password",
  "admin_secret": "secret",
  "apps_domain": "cf.domain",
  "default_security_groups": [ "dns", "public_networks" ],
  "skip_experimental_dynamic_egress_tests": true,
  "skip_ssl_validation": true,
  "test_app_instances": 2,
  "test_applications": 2,
  "proxy_instances": 1,
  "proxy_applications": 1,
  "extra_listen_ports": 2,
  "prefix":"cf-networking-test-"
}
EOF

ginkgo -v $WORKSPACE/cf-networking-release/src/test/acceptance

```

Please refer to [the source code](https://github.com/cloudfoundry/cf-networking-release/blob/develop/src/cf-pusher/config/config.go) for a comprehensive list of accepted configuration parameters.
