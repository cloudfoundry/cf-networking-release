---
title: Sub Projects
expires_at: never
tags: [cf-networking-release]
---

## Sub Projects

- `bosh-dns-adapter`: This job enables `bosh-dns` to resolve requests for internal routes. `bosh-dns` sends DNS requests for internal routes to the `bosh-dns-adapter`. The `bosh-dns-adapter` sends those DNS requests for internal routes to the `service-discovery-controller` to be resolved. Internal domains must be configured on this jobs in addition to being created via CC API.
- `cf-pusher`: This library will help with running `cf push`
- `garden-external-worker`: Also known as the `gaden-cni`. A Garden-RunC / Guardian network plugin that invokes calls out to the CNI. Also forwards ports to support incoming connections from the CF HTTP Router, TCP Router and Diego SSH Proxy.
- `policy-server`: Serves an external facing CRUD API for container-to-container networking polices and dynamic ASG policies. Maintains a database of these policies and the tags associated with each app. | This API serves traffic using TLS. By default serves its external API on port 4002.
- `service-discovery-controller`: Resolves DNS requests for internal routes. It subscribes to internal route updates from NATS. By default listens on port 8054.

