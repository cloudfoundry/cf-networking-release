# netman-release

A [garden-runc](https://github.com/cloudfoundry-incubator/garden-runc-release) add-on
that provides container networking.

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
0. Replace lines with installation procedure for your plugin in this file [`packages/runc-cni/packaging`](https://github.com/cloudfoundry-incubator/netman-release/tree/master/packages/runc-cni/packaging#L11-L14)
	- This will contain the plugin directory where RUNC-CNI will be looking when it is invoking CNI plugins. By default the CNI plugins should end up in `/var/vcap/packages/runc-cni/bin/` on the host VM.
	- For more info on **bosh packaging scripts** read [this](http://bosh.io/docs/packages.html#create-a-packaging-script).

0. Replace flannel specific templates in this directory [`jobs/cni-plugin/templates`](https://github.com/cloudfoundry-incubator/netman-release/tree/master/jobs/cni-plugin/templates)
	- Remove the templates `flannel-watchdog.json.erb`, `flannel-watchdog_ctl.erb`, `flanneld_ctl.erb`.
	- Replace `30-flannel.conf.erb` with the config for your CNI plugin.
	- Change the reference to this file under the `templates` key in the [spec](https://github.com/cloudfoundry-incubator/netman-release/tree/master/jobs/cni-plugin/spec) file.

0. Replace flannel specific config in this file [`jobs/cni-plugin/spec`](https://github.com/cloudfoundry-incubator/netman-release/tree/master/jobs/cni-plugin/spec)
	- Remove lines containing `cni-plugin.flannel` or `flannel-watchdog`
	- For more info on **bosh jobs** read [this](http://bosh.io/docs/jobs.html).
	
0. Make the corresponding changes to your bosh manifest from previous step
	- If you're using the provided manifest generation templates be sure to make the necessary changes.
	- Setting the config in your deployment is done through the deployment manifest [properties](http://bosh.io/docs/deployment-manifest.html#properties).

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
