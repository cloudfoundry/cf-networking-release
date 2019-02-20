# CF Networking Release

CF Networking provides policy-based container networking for Cloud Foundry.

For information about getting started with CF Networking, look at our docs for [the supported mode of deploying to AWS, GCP or BOSH-lite](https://github.com/cloudfoundry/cf-deployment).

## Getting Help

For help or questions with this release or any of its submodules, you can reach the maintainers on Slack at [cloudfoundry.slack.com](https://cloudfoundry.slack.com) in the `#networking` channel.

## Downloads
- Our BOSH release is available [on bosh.io](http://bosh.io/releases/github.com/cloudfoundry-incubator/cf-networking-release)
  and [on our GitHub Releases page](https://github.com/cloudfoundry-incubator/cf-networking-release/releases)

## Documentation
- [Architecture](docs/arch.md)
- Deploy
  - [to AWS, GCP or BOSH-lite](https://github.com/cloudfoundry/cf-deployment)
- Configuring Policies
  - [CLI](docs/CLI.md)
  - [Policy Server API](docs/policy-server-external-api.md)
  - [Policy Server Internal API](docs/policy-server-internal-api.md)
- [Examples](src/example-apps)
  - [Cats & Dogs](https://github.com/cloudfoundry/cf-networking-examples/blob/master/docs/c2c-no-service-discovery.md)
  - [Cats & Dogs With Service Discovery](https://github.com/cloudfoundry/cf-networking-examples/blob/master/docs/c2c-with-service-discovery.md)
  - [Eureka](src/example-apps/eureka)
  - [Proxy](src/example-apps/proxy)
  - [Tick](src/example-apps/tick)
- [3rd Party Plugin Development](docs/3rd-party.md)
- [Contributing to CF Networking](docs/contributing.md)
- [Service Discovery](docs/app-sd.md)
- Operation
  - [Configuration](docs/configuration.md)
  - [Known Issues](docs/known-issues.md)
  - [Troubleshooting](docs/troubleshooting.md)

## Project links
- [Design doc for Container Networking Policy](https://docs.google.com/document/d/1HDS89TJKD7ACG6cqQHph5BdNSKLt8jvo6sPGBZ5DmsM)
- [Engineering backlog](https://www.pivotaltracker.com/n/projects/1498342)
- Chat with us at the `#container-networking` channel on [Cloud Foundry Slack](http://slack.cloudfoundry.org/)
- [CI dashboard](https://c2c.ci.cf-app.com/) and [config](https://github.com/cloudfoundry-incubator/cf-networking-ci)
- [Documentation](./docs)
