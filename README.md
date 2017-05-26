# CF Networking Release

CF Networking provides policy-based container networking for Cloud Foundry.

For information about getting started with CF Networking, look at our docs for [deploying to BOSH-lite](docs/deploy-bosh-lite.md) or [deploying to AWS or GCP](docs/deploy-iaas.md#deploy-to-aws).

## Downloads
- Our BOSH release is available [on bosh.io](http://bosh.io/releases/github.com/cloudfoundry-incubator/cf-networking-release)
  and [on our GitHub Releases page](https://github.com/cloudfoundry-incubator/cf-networking-release/releases)
- Our CF CLI Plugin is [on our GitHub Releases page](https://github.com/cloudfoundry-incubator/cf-networking-release/releases)

## Documentation
- [Architecture](docs/arch.md)
- Deploy
  - [to BOSH-lite](docs/deploy-bosh-lite.md)
  - [to AWS or GCP](docs/deploy-iaas.md)
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
- [CI dashboard](http://dashboard.c2c.cf-app.com) and [config](https://github.com/cloudfoundry-incubator/cf-networking-ci)
- [Documentation](./docs)
