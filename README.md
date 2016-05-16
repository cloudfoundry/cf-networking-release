# netman-release

This release should be deployed so that the `netman` job co-locates with the `garden` job from garden-runc-release.  See below.

## What you can do
- [Running tests](#running-tests)
- [Deploy and test in isolation](#deploy-and-test-in-isolation)
- [Deploying with Diego](#deploying-with-diego)

## Running tests

```bash
docker-machine create --driver virtualbox --virtualbox-cpu-count 4 --virtualbox-memory 2048 dev-box
eval $(docker-machine env dev-box)
~/workspace/netman-release/scripts/docker-test
```

## Deploy and Test in Isolation

```bash
bosh target lite
pushd ~/workspace/garden-runc-release
  git pull
  git submodule sync
  git submodule update --init --recursive
  bosh -n create release
  bosh upload release
popd

pushd ~/workspace/netman-release
  git pull
  git submodule sync
  git submodule update --init --recursive
  bosh -n create release --force && bosh -n upload release
  bosh deployment manifests/netman-manifest.yml
popd

mkdir -p ~/Downloads/releases
pushd ~/Downloads/releases
  curl -L -o consul-release.tgz https://bosh.io/d/github.com/cloudfoundry-incubator/consul-release
  bosh upload release consul-release.tgz
popd

bosh -n deploy
bosh run errand acceptance-tests
```

## Deploying And Testing with Diego

Clone the necessary repositories:

```bash
pushd ~/workspace
  git clone https://github.com/cloudfoundry-incubator/diego-release
  git clone https://github.com/cloudfoundry/cf-release
  git clone https://github.com/cloudfoundry-incubator/netman-release
  git clone https://github.com/cloudfoundry-incubator/garden-runc-release
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
bosh run errand netman-acceptance
```
