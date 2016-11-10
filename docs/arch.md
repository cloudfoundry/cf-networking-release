## Overview

Netman provides policy-driven container networking for Cloud Foundry.

It has several components.  Some are "core" to the platform, others are "swappable" by operators.

![](https://github.com/cloudfoundry-incubator/container-networking-notes/blob/master/container_networking_block_digram.png?raw=true)


### Core components
- [CF CLI plugin](usage.md) enables administrators to control network access policies between CF applications
- Policy Server, a central management node, exposes a JSON REST API used by the CLI plugin
- [Garden External Networker](../src/garden-external-networker), a [Garden-runC](https://github.com/cloudfoundry/garden-runc-release) add-on deployed to every Diego cell
  - Invokes an operator-configured [CNI](https://github.com/containernetworking/cni) Plugin to set up the network for each app instance (container)
  - Forwards ports to support incoming connections from the CF [HTTP Router](https://docs.cloudfoundry.org/concepts/http-routing.html),
    [TCP Router](https://docs.cloudfoundry.org/adminguide/enabling-tcp-routing.html) and [Diego SSH Proxy](https://docs.cloudfoundry.org/concepts/diego/ssh-conceptual.html).
  - Installs egress whitelist rules to support CF [Application Security Groups](https://docs.cloudfoundry.org/adminguide/app-sec-groups.html)

### Batteries included, but swappable
On every Diego cell
- [Flannel](https://github.com/coreos/flannel) [CNI plugin](https://github.com/containernetworking/cni/tree/master/plugins/meta/flannel), provides IP address management and network connectivity to app instances (containers)
  - Uses the flannel [VXLAN backend](https://github.com/coreos/flannel/tree/master/backend/vxlan)
  - Every CF app instance gets a unique IP on a shared, flat L3 network
- VXLAN Policy Agent enforces network policy for network traffic between applications
  - Discovers desired network policies from the [Policy Server's Internal API](3rd-party.md#policy-server-internal-api)
  - Updates IPTables rules on Diego cell to allow whitelisted ingress traffic
  - Egress traffic is tagged with a unique identifier per source application, using the [VXLAN GBP header](https://tools.ietf.org/html/draft-smith-vxlan-group-policy-02#section-2.1)
