# CF Networking

CF Networking provides policy-based container networking for Cloud Foundry.

CF Networking integrates with [Garden-runC](https://github.com/cloudfoundry/garden-runc-release) in a
[Diego](https://github.com/cloudfoundry/diego-release) deployment.  Additionally, a VM is deployed to act as a network Policy Server.
A [JSON API](docs/API.md) and a [CF CLI plugin](docs/CLI.md) are available to control network policies.

For more information about deploying CF Networking, look at our docs for [BOSH-lite](docs/bosh-lite.md) or [AWS](docs/aws.md).

## Downloads
- [BOSH release](http://bosh.io/docs/release.html) available on
  [bosh.io](http://bosh.io/releases/github.com/cloudfoundry-incubator/netman-release)
  and [GitHub Releases](https://github.com/cloudfoundry-incubator/cf-networking/releases)
- [CF CLI Plugin](https://docs.cloudfoundry.org/cf-cli/use-cli-plugins.html) on our [GitHub Releases page](https://github.com/cloudfoundry-incubator/cf-networking/releases)

## Documentation
- [Architecture](docs/arch.md)
- [Deploy to BOSH-lite](docs/bosh-lite.md)
- [Deploy to AWS](docs/aws.md)
- Configuring Policies
  - [CLI](docs/CLI.md)
  - [API](docs/API.md)
- [Examples](src/example-apps)
  - [Cats & Dogs](src/example-apps/cats-and-dogs)
  - [Eureka](src/example-apps/eureka)
  - [Proxy](src/example-apps/proxy)
  - [Tick](src/example-apps/tick)
- [3rd Party Plugin Development](docs/3rd-party.md)
- [Contributing to CF Networking](docs/contributing.md)
- Operation
  - [Configuration](docs/configuration.md)
  - [Known Issues](docs/known-issues.md)
  - [Troubleshooting](docs/troubleshooting.md)

## Project links
- [Design doc for Container Networking Policy](https://docs.google.com/document/d/1HDS89TJKD7ACG6cqQHph5BdNSKLt8jvo6sPGBZ5DmsM)
- [Engineering backlog](https://www.pivotaltracker.com/n/projects/1498342)
- Chat with us at the `#container-networking` channel on [Cloud Foundry Slack](http://slack.cloudfoundry.org/)
- [CI dashboard](http://dashboard.c2c.cf-app.com), [metrics](https://p.datadoghq.com/sb/f3af7f8e2-baf5212773?tv_mode=true) and [config](https://github.com/cloudfoundry-incubator/container-networking-ci)
- [Documentation](./docs)
