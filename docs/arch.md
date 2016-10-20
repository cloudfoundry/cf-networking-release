## Overview

![](https://github.com/cloudfoundry-incubator/container-networking-notes/blob/master/container_networking_block_digram.png?raw=true)

`netman` provides a batteries included container to container system and several APIs for swapping in third party components.
- IPAM and connectivity are provided by a swappable CNI plugin (`flannel` in the batteries included case).
- A swappable policy agent polls garden and the policy server for polices to enforce on the cell. In the provided solution, the VXLAN policy agent writes iptables rules to filter packets based on VXLAN gbp tags.
- Inbound traffic from the gorouter is port forwarded from the cell to the container via a NetIn rule. NetIn calls are made by garden to the external networker which then writes the iptables NAT rule.
- Application security groups are enforced by NetOut calls from garden. The external networker also writes iptables rules to enforce ASGs.


