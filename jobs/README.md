# CF Networking Release Jobs

This is the README for CF-Networking-Release jobs. To learn more about `cf-networking-release`, go to the main [README](../README.md).

| Job Name | Purpose | Additional Notes |
| --- | --- | --- |
| bbr-cfnetworkingdb | Allows operator to opt-in to BOSH backup and restore the policy-server database. |  |
| bosh-dns-adapter| This job enables `bosh-dns` to resolve requests for internal routes. `bosh-dns` sends DNS requests for internal routes to the `bosh-dns-adapter`. The `bosh-dns-adapter` sends those DNS requests for internal routes to the `service-discovery-controller` to be resolved. | Internal domains must be configured on this jobs in addition to being created via CC API. |
| garden-cni | Also known as the `Garden External Networker`. A Garden-RunC / Guardian network plugin that invokes calls out to the CNI. Also forwards ports to support incoming connections from the CF HTTP Router, TCP Router and Diego SSH Proxy. |  |
| performance-test-sd | Runs service discovery performance tests.  | Used exclusively in CI by `mitre-perf-tests` |
| policy-server | Serves an external facing CR\*D API for container-to-container networking polices and dynamic ASG policies. Maintains a database of these policies and the tags associated with each app. | This API serves traffic using TLS. By default serves its external API on port 4002. |
| policy-server-internal | Serves the internal API used by the `vxlan-policy-agent` to discover container-to-container networking polices and dynamic ASG policies.  | The internal API traffic is secured via mutual TLS. By default serves its internal API on port 4003. |
| service-discovery-controller | Resolves DNS requests for internal routes. It subscribes to internal route updates from NATS. | By default listens on port 8054. |
