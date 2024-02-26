# CF Networking Acceptance Tests

Finally! A guide for how to run CF Networking Acceptance Tests!

## Environment Prerequisites

- If not enabling `skip_search_domain_tests`, the `apps.internal` domain must be configured as a search domain in `garden-cni`. See [this ops-file](https://github.com/cloudfoundry/wg-app-platform-runtime-ci/blob/main/cf-networking-release/opsfiles/add-apps-internal-search-domain.yml).
- `dynamic_asgs_enabled` must match whether or not the environment has dynamic ASGs enabled or disabled.
- If enabling `run_experimental_outbound_conn_limit_test`, ensure silk-cni is configured to rate limit egress traffic via [this ops file](https://github.com/cloudfoundry/wg-app-platform-runtime-ci/blob/main/cf-networking-release/opsfiles/limit-app-outbound-connections.yml).

## Configuration Parameters

- `api` - CF environment's API endpoint
- `apps_domain` - The domain that apps should be pushed to when being tested. Must exist in DNS + as a shared-domain on the CF environment
- `admin_user` - Admin username for authenticating to CF
- `admin_password` - Admin password for authenticating to CF
- `admin_secret` - Admin user's UAA client secret
- `asg_size` - Not used in acceptance tests. Used for ASG performance tests.
- `test_app_instances` - Number of app instances to scale to when testing C2C policies between applications
- `test_app_registy_ttl_seconds` - The value of the `REGISTRY_TTL_SECONDS` env var given to the `ticker` app for its tick registration TTL
- `test_applications` - Number of applications to push when testing C2C policies between applications
- `concurrency` - Number of threads to do when performing expensive CF API tasks like interacting with the apps API
- `default_security_groups` - Names of the default running (not staging) security groups that are applied to apps in the CF environment. This is used to bind/undbind the default ASGs when needing to test specific traffic patterns in isolation.
- `dynamic_asgs_enabled` - Configures ASG related tests to behave appropriately based on whether or not the environment under test has Dynamic ASGs enabled
- `extra_listen_ports` - Number of additional ports for test apps to listen on when testing C2C policies (basic is just 8080, additional ports start at 7000).
- `internetless` - Disables tests requiring external connectivity, if the environment under test has no internet access
- `policy_update_wait_seconds` - Not used in acceptance tests. Used in scaling tests.
- `prefix` - prefix string to prepend CF resources with for the tests
- `proxy_applications` - number of `proxy` applications to deploy in the inter-container connectivity tests
- `proxy_instances` - number of AIs to scale `proxy`applications to in the inter-container connectivity tests
- `sample_percent` - Unused in CF Networking Acceptance tests (used in scaling tests, which share the config datastructure)
- `skip_icmp_tests` - Disables ICMP egress tests, useful for environments where ICMP traffic is blocked at an infrastructure level.
- `run_custom_iptables_compatibility_test` - Enables/Disables testing the compatibility of custom iptables rules on diego cells with iptables rules managed by vxlan-policy-agent
- `run_experimental_outbound_conn_limit_test`- Enables/Disables testing of the experimental option for outbound connection rate limiting.
- `skip_space_developer_policy_test` - Unused.
- `skip_search_domain_tests` - Disables/Enables tests to validate that search domains are propagated into the app containers' `/etc/resolv.conf`
- `skip_ssl_validation` - Enables/Disables SSL validation when talking to CF + apps

## Recommended Config Options

```
{
    "admin_password": "CF_ADMIN_PASSWORD",
    "admin_secret": "CF_ADMIN_UAA_SECRET",
    "admin_user":"admin",
    "api": "api.CF_SYSTEM_DOMAIN},
    "apps_domain": "CF_APPS_DOMAIN",
    "concurrency": 16,
    "default_security_groups": [ "dns", "public_networks" ],
    "dynamic_asgs_enabled": true,
    "extra_listen_ports": 2,
    "internetless": false,
    "nodes": 1,
    "prefix":"test-",
    "proxy_applications": 1,
    "proxy_instances": 1,
    "run_custom_iptables_compatibility_test": true,
    "run_experimental_outbound_conn_limit_test": true,
    "skip_icmp_tests": false,
    "skip_search_domain_tests": false,
    "skip_ssl_validation":true,
    "test_app_instances": 3,
    "test_applications": 2,
    "test_app_registry_ttl_seconds": 10,
    "include_security_groups": true,
    "use_http":true
}
```

## Known Issues

- Tests involving the `tick` app have been known to be flakey. Retrying the test suite will usually resolve the issue.
