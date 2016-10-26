# netman-release

A [Garden-runC](https://github.com/cloudfoundry/garden-runc-release) add-on that provides container networking for CloudFoundry.

## Overview

`netman` provides a batteries included container to container system and several APIs for swapping in third party components.
- IPAM and connectivity are provided by a swappable CNI plugin (`flannel` in the batteries included case).
- A swappable policy agent polls garden and the policy server for polices to enforce on the cell. In the provided solution, the VXLAN policy agent writes iptables rules to filter packets based on VXLAN gbp tags.
- Inbound traffic from the gorouter is port forwarded from the cell to the container via a NetIn rule. NetIn calls are made by garden to the external networker which then writes the iptables NAT rule.
- Application security groups are enforced by NetOut calls from garden. The external networker also writes iptables rules to enforce ASGs.

# Documentation

- [Architecture](docs/arch.md)
- [Deploy to BOSH-lite](docs/bosh-lite.md)
- [Deploy to AWS](docs/aws.md)
- [Configuring Policies - CLI and API](docs/usage.md)
- [Examples](src/example-apps)
  - [Cats & Dogs](src/example-apps/cats-and-dogs)
  - [Proxy](src/example-apps/proxy)
  - [Tick](src/example-apps/tick)
- [3rd Party Plugin Development](docs/3rd-party.md)
- [Contributing to Netman](docs/contributing.md)
- [Operation](docs/operation.md)
- [Known Issues](docs/known-issues.md)

## Project links
- [Design doc for Container Networking Policy](https://docs.google.com/document/d/1HDS89TJKD7ACG6cqQHph5BdNSKLt8jvo6sPGBZ5DmsM)
- [Engineering backlog](https://www.pivotaltracker.com/n/projects/1498342)
- Chat with us at the `#container-networking` channel on [CloudFoundry Slack](http://slack.cloudfoundry.org/)
- [CI dashboard](http://dashboard.c2c.cf-app.com) and [config](https://github.com/cloudfoundry-incubator/container-networking-ci)
- [Documentation](./docs)
