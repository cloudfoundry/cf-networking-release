# Test Overview

### Unit, integration and template tests

To run everything:
```bash
cd cf-networking-release
./scripts/docker-test
./scripts/template-tests
```

To run individual tests during development:

```bash
cd cf-networking-release
./scripts/docker-shell-with-started-db
# now use ginkgo to run whatever tests you want
```

### Acceptance tests
Acceptance tests require a fully deployed Cloud Foundry. 

⚠️ Warning: these tests remove default security groups. Do not run these tests on a prod environment.

1. Make the following config.yml and fill in the properties
```bash
{
  "api": "API_URL",
  "admin_user": "admin",
  "admin_password": "ADMIN_PASSWORD", # this should be in credhub as "cf_admin_password"
  "admin_secret": "ADMIN_SECRET", # this should be in credhub as "uaa_admin_client_secret"
  "apps_domain": "APPS_DOMAIN",
  "default_security_groups": [ "dns", "public_networks" ], # check these against your own security groups.
  "skip_experimental_dynamic_egress_tests": false,
  "skip_ssl_validation": true,
  "test_app_instances": 2,
  "test_applications": 2,
  "proxy_instances": 1,
  "proxy_applications": 1,
  "extra_listen_ports": 2,
  "prefix":"cf-networking-test-app"
}
```

2. Run the following command for c2c acceptance tests:
```
CONFIG=config.yml APPS_DIR=${PWD}/src/example-apps ginkgo -v
src/code.cloudfoundry.org/test/acceptance

```

3. Run the following command for service discovery acceptance tests:
```
CONFIG=config.yml APPS_DIR=${PWD}/src/example-apps ginkgo -v src/code.cloudfoundry.org/test/acceptance-sd
```

#### Foundation configuration tips & tricks

The following ops files are required to ensure a successful run:

- [add-apps-internal-search-domain.yml](https://github.com/cloudfoundry/cf-networking-release/blob/develop/manifest-generation/opsfiles/add-apps-internal-search-domain.yml)
- [enable-experimental-dynamic-egress-policies.yml](https://github.com/cloudfoundry/cf-networking-release/blob/develop/manifest-generation/opsfiles/enable-experimental-dynamic-egress-policies.yml)
- [disable-ingress-redirect-to-proxy.yml](https://github.com/cloudfoundry/cf-networking-release/blob/develop/ci/opsfiles/disable-ingress-redirect-to-proxy.yml)
- [scale-to-2-diego-cells.yml](https://github.com/cloudfoundry/cf-networking-release/blob/develop/ci/opsfiles/scale-to-2-diego-cells.yml)

⚠️ The `vxlan-policy-agent.properties.policy_poll_interval_seconds` should be <= 5.

##### Running the app outbound connection rate limit test

This test is disabled by default and could be enabled by setting `run_experimental_outbound_conn_limit_test` to `true` as part of the test config.yml above.
Additionally the [limit-app-outbound-connections.yml](https://github.com/cloudfoundry/cf-networking-release/blob/develop/ci/opsfiles/limit-app-outbound-connections.yml) ops file is required to properly configure and enable the connection rate limiting feature.
