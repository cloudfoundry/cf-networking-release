# netman-release

A [garden-runc](https://github.com/cloudfoundry-incubator/garden-runc-release) add-on
that provides container networking.

## Project links
- [Design doc for Container Networking Policy](https://docs.google.com/document/d/1HDS89TJKD7ACG6cqQHph5BdNSKLt8jvo6sPGBZ5DmsM)
- [Engineering backlog](https://www.pivotaltracker.com/n/projects/1498342)
- Chat with us at the `#container-networking` channel on [CloudFoundry Slack](http://slack.cloudfoundry.org/)

## What you can do
- [Running tests](#running-tests)
- [Deploy and test in isolation](#deploy-and-test-in-isolation)
- [Deploy and test with Diego](#deploy-and-test-with-diego)
- [Using your own CNI plugin](#using-your-own-cni-plugin)

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

## Deploy and test in isolation

```bash
bosh target lite

cd ~/workspace/netman-release

./scripts/update
bosh -n create release --force && bosh -n upload release --rebase
bosh deployment bosh-lite/deployments/netman-bare.yml

bosh -n deploy
bosh run errand acceptance-tests
```

## Using your own CNI plugin
**Note: these instructions say to put 3rd party plugins into a job called `cni-flannel`.  Feel free to rename that job if you're no longer using flannel.**

0. Replace lines with installation procedure for your plugin in this file [`packages/runc-cni/packaging`](https://github.com/cloudfoundry-incubator/netman-release/tree/master/packages/runc-cni/packaging#L11-L14)
	- This will contain the plugin directory where RUNC-CNI will be looking when it is invoking CNI plugins. By default the CNI plugins should end up in `/var/vcap/packages/runc-cni/bin/` on the host VM.
	- For more info on **bosh packaging scripts** read [this](http://bosh.io/docs/packages.html#create-a-packaging-script).

0. Remove the `packages/flannel/` and `packages/flannel-watchdog/` directories

0. Replace flannel specific templates in this directory [`jobs/cni-flannel/templates`](https://github.com/cloudfoundry-incubator/netman-release/tree/master/jobs/cni-flannel/templates)
	- Remove the templates `flannel-watchdog.json.erb`, `flannel-watchdog_ctl.erb`, `flanneld_ctl.erb`.
	- Replace `30-flannel.conf.erb` with the config for your CNI plugin.

0. Remove the contents of this file [`jobs/cni-flannel/monit`](https://github.com/cloudfoundry-incubator/netman-release/tree/master/jobs/cni-flannel/monit)


0. Replace flannel specific config in this file [`jobs/cni-flannel/spec`](https://github.com/cloudfoundry-incubator/netman-release/tree/master/jobs/cni-flannel/spec)
	- Change the reference to the `30-flannel.conf.erb` file under the `templates` key.
	- Remove `flannel` and `flannel-watchdog` under the `packages` key.
	- Remove lines containing `cni-flannel.flannel`, `cni-flannel.etcd_endpoints` or `flannel-watchdog` under the `properties` key.
	- For more info on **bosh jobs** read [this](http://bosh.io/docs/jobs.html).
0. Remove the references to `flannel` and `flannel-watchdog` from [`scripts/sync-package-specs`](https://github.com/cloudfoundry-incubator/netman-release/tree/master/scripts/sync-package-specs#L42-L46)

0. Make the corresponding config changes to your bosh manifest
	- Setting the config in your deployment is done through the deployment manifest [properties](http://bosh.io/docs/deployment-manifest.html#properties).
	- If you're using the provided manifest generation templates be sure to make the necessary changes.
	- If you want to deploy using the [netman-bare](https://github.com/cloudfoundry-incubator/netman-release/blob/master/bosh-lite/deployments/netman-bare.yml) manifest, just add any config specified in `jobs/cni-flannel/spec` under the `properties` key.


See [here](https://gist.github.com/jaydunk/97ddf7c3a9384ca76f1b9d8bb1a92d0b) for an example patch which removes flannel and replaces it with the bridge cni plugin.

###Installing the plugin
To install your CNI plugin you will need to add the executable to the bosh packaging script in [`packages/runc-cni/packaging`](https://github.com/cloudfoundry-incubator/netman-release/tree/master/packages/runc-cni/packaging). This will contain the plugin directory where RUNC-CNI will be looking when it is invoking CNI plugins. By default the CNI plugins should end up in `/var/vcap/packages/runc-cni/bin/` on the host VM.

For more info on **bosh packaging scripts** read [this](http://bosh.io/docs/packages.html#create-a-packaging-script)

###Getting the CNI config
This requires modifications to the cni-flannel [job](https://github.com/cloudfoundry-incubator/netman-release/tree/master/jobs/cni-flannel). The cni-flannel job contains the config for the netman-agent CNI handler as well as some flannel specific config. To use your own plugin remove all the flannel specific config (fields beginning with `cni-flannel.flannel` or `flannel-watchdog`) from the spec file `jobs/cni-flannel/spec` and replace with your plugin specific config.

In the templates directory, include whatever config file is needed for your CNI plugin. Then under the templates key in the spec file make sure that the config file ends up in `config/cni/<your-config>.conf`.

For more info on **bosh jobs** read [this](http://bosh.io/docs/jobs.html).

The properties specified in the spec file are configured via the bosh deployment manifest [properties](http://bosh.io/docs/deployment-manifest.html#properties).

## Testing the policy server
To accept:

```
cf auth network-admin network-admin
cf curl /networking/v0/external/policies
```

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
