# CF Networking Release

This repository is a [BOSH](https://github.com/cloudfoundry/bosh) release for deploying CF Networking and associated tasks. CF Networking provides policy-based container networking and service discovery for Cloud Foundry.

For information on getting started with Cloud Foundry look at the docs for
[CF Deployment](https://github.com/cloudfoundry/cf-deployment).

## Downloads

Our BOSH release is available [on bosh.io](http://bosh.io/releases/github.com/cloudfoundry-incubator/cf-networking-release)
  and [on our GitHub Releases page](https://github.com/cloudfoundry-incubator/cf-networking-release/releases)

## Getting Help

For help or questions with this release or any of its submodules, you can reach
the maintainers on Slack at
[cloudfoundry.slack.com](https://cloudfoundry.slack.com) in the `#cf-for-vms-networking`
channel.


## Contributing

Please look at the ["Contributing to CF Networking" doc](docs/contributing.md) for more information.

## Docs Table of Contents

1. [CF Networking Operator Resources](#cf-networking-operator-resources)
1. [CF App Developer Resources](#cf-networking-app-dev-resources)
1. [CF Networking Contributor Resources](#contributor-resources)
1. [CF CNI Plugin Developer Resources](#cni-plugin-dev-resources)

---

## <a name="cf-networking-operator-resources"></a>CF Networking Operator Resources

### <a name="what-is-cf-networking"></a>What does CF Networking do?

- [What is CF Networking](docs/what-is-cf-networking.md) explains the problems
  that this release solves and describes the basic functionaltiy that this
  release provides.

### <a name="cf-networking-architecture"></a>CF Networking Architecture

- [Container to Container Networking Architecture](docs/arch.md) goes step-by-step through the container to container networking control plane and data plane flows.

- [Service Discovery Architecture](docs/service-discovery-architecture.md) goes step-by-step through the control plane and data plane flows for service discovery.

### <a name="deploying-cf-networking"></a>Deploying CF Networking

CF Networking is automatically included in [CF Deployment](https://github.com/cloudfoundry/cf-deployment). You don't have to do anything!

### <a name="configuring-cf-networking"></a>Configuring CF Networking

- [Configuring Container to Container Networking](docs/configuring-c2c.md) describes some of the most popular ways to configure container to container networking and its policies. 
- [Configuring Service Discovery](docs/configuring-sd.md) describes some of the most popular ways to configure service discovery, including how to create more internal domains.


### <a name="monitoring-cf-networking"></a>Monitoring

- [Service Discovery Metrics](docs/service-discovery-metrics.md) lists all of the available metrics for the service discovery components.
- TODO - add doc for c2c networking metrics.

### <a name="troubleshooting-cf-networking"></a>Troubleshooting

- [Overlay Network Troubleshooting](docs/troubleshooting-container-overlay.md) describes general debugging tips for overlay networking and common failures.
- [Container to Container Networking Troubleshooting](docs/troubleshooting.md) describes general debugging tips for c2c networking and some specific common failures.
- [Debugging Latency with Container to Container Networking](docs/network-latency-troubleshooting.md) describes how to determine the root cause for c2c latency.
- [Network Policy Database Overview](docs/network-policy-database-overview.md)
- [Known Issues](docs/known-issues.md) contains a list of known bugs for c2c networking. This list has not been maintained since 2018 and even then it was only sporadically maintained.


## <a name="cf-networking-app-dev-resources"></a>CF Networking App Developer Resources

### <a name="using-cf-networking-with-clis"></a>Using CF Networking with CLIs

- [CF Docs for "Configuring Container-to-Container Networking"](https://docs.cloudfoundry.org/devguide/deploy-apps/cf-networking.html#-create-and-manage-networking-policies
) contains information on how app developers can create/list/delete network policies.
- [CF Docs for "Configuring Routes and Domains"](https://docs.cloudfoundry.org/devguide/deploy-apps/routes-domains.html#internal-routes)
contains information on how app developers can create internal routes for c2c networking.

### <a name="using-cf-networking-apis"></a>Using CF Networking APIs

- [Policy Server External API](docs/policy-server-external-api.md) contains full docs on all endpoints for the external policy server. This includes creating, listing, and deleting policies. 

- [Policy Server Internal API](docs/policy-server-internal-api.md) contains full docs on all endpoints for the internal policy server. This includes creating and listing tags and getting policies. 

### <a name="example-apps"></a>Example Apps

- [Example Apps Overview](docs/example-apps-overview.md) describes a handful of example apps (like the ever popular proxy) and the features that they provide.


## <a name="contributor-resources"></a>CF Networking Contributor Resources
### <a name="running-tests"></a>Contributing Guide

- The [Contributing Guide](docs/contributing.md) describes the steps you should take to contribute. Thanks in advance! We love our community :D 
- [Adding Libraries or Packages](docs/adding-libraries-or-packages.md) describes how to add external golang libraries or new bosh packages to this release.
### <a name="running-tests"></a>Running Tests

- [Test Overview](docs/test-overview.md) describes the many tests for CF Networking and how to run them. Running these tests is a requirement for contributors.

## <a name="cni-plugin-dev-resources"></a>CF CNI Plugin Developer Resources

- [3rd Party Plugin Development](docs/3rd-party.md) describes how to create a CNI plugin for CF that would replace silk-release.
