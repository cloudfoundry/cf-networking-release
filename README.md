# netman-release

A [garden-runc](https://github.com/cloudfoundry-incubator/garden-runc-release) add-on
that provides container networking.

## What you can do
- [Running tests](#running-tests)
- [Deploy and test in isolation](#deploy-and-test-in-isolation)
- [Deploy and test with Diego](#deploy-and-test-with-diego)

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
