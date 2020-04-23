# CF Networking Release Jobs

This is the README for CF-Networking-Release jobs. To learn more about `cf-networking-release`, go to the main [README](../README.md).

| Job Name | Purpose | Additional Notes |
| --- | --- | --- |
| bbr-cfnetworkingdb | Allows operator to opt-in to BOSH backup and and restore the policy-server. |  |
| bosh-dns-adapter| Installs BOSH DNS Adapter on a Diego cell so apps running can establish c2c communication through a known route served by internal BOSH DNS.  | Allows to enable/disable c2c service discovery. |
| garden-cni | Also known as the `Garden External Networker`. A Garden-RunC / Guardian network plugin that invokes calls out to the CNI. Also forwards ports to support incoming connections from the CF HTTP Router, TCP Router and Diego SSH Proxy. |  |
| performance-test-sd | Runs NATS performance tests.  | Used exclusively in CI by `mitre-perf-tests` |
| policy-server | Central management node. Maintains a database of policies for traffic between apps.Serves an externally-facing API for creating, deleting and listing network polices and tags in the policy database. | This API serves traffic using TLS. By default serves its external API on port 4002. |
| policy-server-internal | Similar to `policy-server` but has an Internal API. The Policy Server Internal is used VXLAN Policy Agent to discover desired network policies.  | The internal API traffic is secured via  mutual TLS. By default serves its internal API on port 4003. |
| service-discovery-controller | Deploys the Service Discovery Controller which subscribes to route updates from NATS on the internal routes. | By default listens on port 8054. Traffic secured via TLS. |
