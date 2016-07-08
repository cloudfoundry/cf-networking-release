# netman-release

A [garden-runc](https://github.com/cloudfoundry-incubator/garden-runc-release) add-on
that provides container networking.

## Project links
- [Design doc for Container Networking Policy](https://docs.google.com/document/d/1HDS89TJKD7ACG6cqQHph5BdNSKLt8jvo6sPGBZ5DmsM)
- [Engineering backlog](https://www.pivotaltracker.com/n/projects/1498342)
- Chat with us at the `#container-networking` channel on [CloudFoundry Slack](http://slack.cloudfoundry.org/)

## Running tests

```bash
docker-machine create --driver virtualbox --virtualbox-cpu-count 4 --virtualbox-memory 2048 dev-box
eval $(docker-machine env dev-box)
~/workspace/netman-release/scripts/docker-test
```

## Deploy and test with Diego

Clone the necessary repositories

```bash
pushd ~/workspace
  git clone https://github.com/cloudfoundry-incubator/diego-release
  git clone https://github.com/cloudfoundry/cf-release
  git clone https://github.com/cloudfoundry-incubator/netman-release
popd
```

Run the deploy script

```bash
pushd ~/workspace/netman-release
  ./scripts/deploy-to-bosh-lite
popd
```

Finally, run the acceptance errand:

```bash
bosh run errand netman-cf-acceptance
```

## Testing the policy server
To accept:

```bash
cf auth network-admin network-admin

# list policies
cf curl /networking/v0/external/policies

# create a new policy
cf curl -X POST /networking/v0/external/policies -d '{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8080 } } ] }'

# delete that policy
cf curl -X DELETE /networking/v0/external/policies -d '{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8080 } } ] }'

```

## Using your own CNI plugin
0. Remove the following BOSH jobs:
  - `cni-flannel`
  - `netman-agent`
0. Remove the following BOSH packages:
  - `flannel`
  - `flannel-watchdog`
  - `netman-agent`
0. Add in all packages and jobs required by your CNI plugin.  At a minimum, you must provide a CNI binary program and a CNI config file.
  - For more info on **bosh packaging scripts** read [this](http://bosh.io/docs/packages.html#create-a-packaging-script).
  - For more info on **bosh jobs** read [this](http://bosh.io/docs/jobs.html).
0. Update the [deployment manifest properties](http://bosh.io/docs/deployment-manifest.html#properties):
  ```yaml
  garden-cni:
    adapter:
      cni_plugin_dir: /var/vcap/packages/YOUR_PACKAGE/bin # your CNI binary goes in this directory
      cni_config_dir: /var/vcap/jobs/YOUR_JOB/config/cni  # your CNI config file goes in this directory
  ```
  Remove any lingering references to `flannel` or `cni-flannel` in the deployment manifest.


# Development

### Referencing a new library from existing BOSH package
1. Add any new libraries into the submodule from the root of the repo
  ```
  cd $GOPATH
  git submodule add https://github.com/foo/bar src/github.com/foo/bar
  ./scripts/sync-package-specs
  ```

### Adding a new BOSH package
1. Add any new libraries into the submodules from the root of the repo
  ```
  cd $GOPATH
  git submodule add https://github.com/foo/bar src/github.com/foo/bar
  ```

2. Update the package sync script:
  ```
  vim $GOPATH/scripts/sync-package-specs
  ```
  Find or create the `sync_package` line for `baz`

3. Run the sync script:
  ```
  ./scripts/sync-package-specs
  ```
