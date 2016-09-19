# netman-release

A [garden-runc](https://github.com/cloudfoundry/garden-runc-release) add-on
that provides container networking.

## Overview

![](https://github.com/cloudfoundry-incubator/container-networking-notes/blob/master/container_networking_block_digram.png?raw=true)

`netman` provides a batteries included container to container system and several APIs for swapping in third party components.
- IPAM and connectivity are provided by a swappable CNI plugin (`flannel` in the batteries included case).
- A swappable policy agent polls garden and the policy server for polices to enforce on the cell. In the provided solution, the VXLAN policy agent writes iptables rules to filter packets based on VXLAN gbp tags.
- Inbound traffic from the gorouter is port forwarded from the cell to the container via a NetIn rule. NetIn calls are made by garden to the external networker which then writes the iptables NAT rule.
- Application security groups are enforced by NetOut calls from garden. The external networker also writes iptables rules to enforce ASGs.

## Project links
- [Design doc for Container Networking Policy](https://docs.google.com/document/d/1HDS89TJKD7ACG6cqQHph5BdNSKLt8jvo6sPGBZ5DmsM)
- [Engineering backlog](https://www.pivotaltracker.com/n/projects/1498342)
- Chat with us at the `#container-networking` channel on [CloudFoundry Slack](http://slack.cloudfoundry.org/)
- [CI dashboard](http://dashboard.c2c.cf-app.com) and [config](https://github.com/cloudfoundry-incubator/container-networking-ci)

## NOTICE
`netman` is currently only compatible with the `develop` branch of `garden-runc-release`.

## Install the CF CLI Plugin

1. Get the binary

  - Option 1: Download a precompiled binary of the `network-policy-plugin` for your operating system from our [GitHub Releases](https://github.com/cloudfoundry-incubator/netman-release/releases)

  - Option 2: Build from source

    ```bash
    go build -o /tmp/network-policy-plugin ./src/cli-plugin
    ```

2. Install it

  ```bash
  chmod +x ~/Downloads/network-policy-plugin
  cf install-plugin ~/Downloads/network-policy-plugin
  ```

## Deploy to bosh-lite

Follow the instructions [here](https://github.com/cloudfoundry/bosh-lite) to install `bosh-lite` on your machine.

Ensure that `br_netfilter` is enabled on your vagrant box:
```bash
pushd ~/workspace/bosh-lite
  vagrant ssh -c 'sudo modprobe br_netfilter'
popd
```

Upload the latest `bosh-lite` stemcell 
```
bosh upload stemcell https://bosh.io/d/stemcells/bosh-warden-boshlite-ubuntu-trusty-go_agent
```

Or download the stemcell and manually upload it to `bosh-lite` (potentially faster)
```
curl -L -o bosh-lite-stemcell-latest.tgz https://bosh.io/d/stemcells/bosh-warden-boshlite-ubuntu-trusty-go_agent
bosh upload stemcell bosh-lite-stemcell-latest.tgz
```

Then grab the required releases
```
pushd ~/workspace
  git clone https://github.com/cloudfoundry/diego-release
  git clone https://github.com/cloudfoundry/cf-release
  git clone https://github.com/cloudfoundry-incubator/netman-release
popd
```

**Note:** The current netman release depends on `develop` branch of garden-runc-release, please do the following:
```
pushd ~/workspace
  git clone https://github.com/cloudfoundry/garden-runc-release
  cd garden-runc-release
  git checkout develop
  git submodule update --init --recursive
  bosh target lite && bosh create release && bosh upload release
popd
```

and deploy
```
pushd ~/workspace/netman-release
  ./scripts/deploy-to-bosh-lite
popd
```

Then follow [the instructions for testing with the cats & dogs example](https://github.com/cloudfoundry-incubator/netman-release/tree/master/src/example-apps/cats-and-dogs).


## Deploy to AWS
0. Upload stemcell with Linux kernel 4.4 to bosh director.  Versions >= 3262.5 should work.
0. Create netman stubs
  - netman requires additional information in several stubs.
  - Add under `properties: uaa` in `stubs/cf/properties.yml`:

    ```
    scim:
      users:
      - admin|<admin-password>|scim.write,scim.read,openid,cloud_controller.admin,doppler.firehose,network.admin
    clients:
      cf:
        scope: cloud_controller.read,cloud_controller.write,openid,password.write,cloud_controller.admin,scim.read,scim.write,doppler.firehose,uaa.user,routing.router_groups.read,network.admin
    ```

  - Add under `properties` in `stubs/cf/properties.yml`:

    ```
    acceptance_tests:
      admin_password: <admin-password>
      admin_user: admin
      api: api.<system-domain>
      apps_domain: <apps-domain>
      nodes: 1
      skip_ssl_validation: true
      use_http: true
    ```

  - Create a `cf_creds_stub.yml`

    ```
    ---
    properties:
      uaa:
        clients:
          network-policy:
            secret: <uaa-client-secret>
      policy-server:
        connection_string: <db-connection-string>
    ```

0. Generate diego with netman manifest
  - Run `generate-deployment-manifest`. Set `environment_path` to the directory containing your stubs for cf, diego, and netman.
    Set `output_path` to the directory you want your manifest to be created in.
    Set `diego_release_path` to your local copy of the diego-release repository.

  ```bash
  set -e -x -u

  environment_path=
  output_path=
  diego_release_path=

  pushd cf-release
    ./scripts/generate_deployment_manifest aws \
      ${environment_path}/stubs/director-uuid.yml \
      ${diego_release_path}/examples/aws/stubs/cf/diego.yml \
      ${environment_path}/stubs/cf/properties.yml \
      ${environment_path}/stubs/cf/instance-count-overrides.yml \
      ${environment_path}/stubs/cf/stub.yml \
      > ${output_path}/cf.yml
  popd

  pushd diego-release
    ./scripts/generate-deployment-manifest \
      -g \
      -c ${output_path}/cf.yml \
      -i ${environment_path}/stubs/diego/iaas-settings.yml \
      -p ${environment_path}/stubs/diego/property-overrides.yml \
      -n ${environment_path}/stubs/diego/instance-count-overrides.yml \
      -v ${environment_path}/stubs/diego/release-versions.yml \
      > ${output_path}/diego0.yml
  popd

  pushd netman-release
    ./scripts/netmanify \
      ${output_path}/diego0.yml \
      ${environment_path}/stubs/netman/cf_creds_stub.yml \
      ${environment_path}/stubs/cf/stub.yml \
      > ${output_path}/diego.yml
  popd
  ```

0. Deploy
  - Target your bosh director.
  ```
  bosh target <your-director>
  ```
  - Set the deployment
  ```
  bosh deployment ${output_path}/diego.yml
  ```
  - Deploy
  ```
  bosh deploy
  ```

## Other infrastructures
We do not currently test this software on infrastructures other than BOSH-lite and AWS.  With recent stemcells and the appropriate manifest changes, it should work.  Let us know if you find issues.


## To replace flannel with your own CNI plugin
0. Remove the following BOSH jobs:
  - `cni-flannel`
  - `vxlan-policy-agent`
0. Remove the following BOSH packages:
  - `flannel`
  - `flannel-watchdog`
0. Add in all packages and jobs required by your CNI plugin.  At a minimum, you must provide a CNI binary program and a CNI config file.
  - For more info on **bosh packaging scripts** read [this](http://bosh.io/docs/packages.html#create-a-packaging-script).
  - For more info on **bosh jobs** read [this](http://bosh.io/docs/jobs.html).
0. Update the [deployment manifest properties](http://bosh.io/docs/deployment-manifest.html#properties)

  ```yaml
  garden-cni:
    adapter:
      cni_plugin_dir: /var/vcap/packages/YOUR_PACKAGE/bin # your CNI binary goes in this directory
      cni_config_dir: /var/vcap/jobs/YOUR_JOB/config/cni  # your CNI config file goes in this directory
  ```
  Remove any lingering references to `flannel` or `cni-flannel` in the deployment manifest.

## To deploy a local-only (no-op) CNI plugin
As a baseline, you can deploy using only the basic [bridge CNI plugin](https://github.com/containernetworking/cni/blob/master/Documentation/bridge.md).

This plugin will provide connectivity between containers on the same Garden host (Diego cell)
but will not provide a cross-host network.  However, it can be a useful baseline configuration for
testing and development.

```bash
cd bosh-lite
bosh target lite
bosh update cloud-config cloud-config.yml
bosh deployment local-only.yml
bosh deploy
```

## To deploy diego with CNI but without cross-host container networking
For generating a cloudfoundry-diego deployment without container to container connectivity, but using the CNI bridge plugin for NAT'ed connectivity.

```bash
CNI_BRIDGE=true ./scripts/generate-bosh-lite-manifests
bosh deploy
```

## Development

### Running low-level tests

```bash
~/workspace/netman-release/scripts/docker-test
```

### Running the full acceptance test on bosh-lite
WARNING: This test is taxing and has an aggressive timeout.
It may fail on a laptop or other underpowered bosh-lite.

```bash
cd src/netman-cf-acceptance
./run-locally.sh
```

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

## Syslog forwarding
To forward all logs from your bosh-lite to a syslog destination (like Papertrail),
add the following block to `manifest-generation/stubs/bosh-lite-cf.yml`:
```yaml
  syslog_daemon_config:
    address: some-syslog-host.example.com
    port: 12345
    transport: udp
```
