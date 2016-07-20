# netman-release

A [garden-runc](https://github.com/cloudfoundry-incubator/garden-runc-release) add-on
that provides container networking.

## Project links
- [Design doc for Container Networking Policy](https://docs.google.com/document/d/1HDS89TJKD7ACG6cqQHph5BdNSKLt8jvo6sPGBZ5DmsM)
- [Engineering backlog](https://www.pivotaltracker.com/n/projects/1498342)
- Chat with us at the `#container-networking` channel on [CloudFoundry Slack](http://slack.cloudfoundry.org/)
- [CI pipeline](https://c2c.ci.cf-app.com/) and [config](https://github.com/cloudfoundry-incubator/container-networking-ci)


## Install the CF CLI Plugin

1. Get the binary

  - Option 1: Download a precompiled binary for Mac from our [GitHub Releases](https://github.com/cloudfoundry-incubator/netman-release/releases)

  - Option 2: Build from source

    ```bash
    go build -o /tmp/network-policy-plugin ./src/cli-plugin
    ```

2. Install it

  ```bash
  chmod +x ~/Downloads/network-policy-plugin
  cf install-plugin ~/Downloads/network-policy-plugin
  ```

## Deploy netman-release and test

```bash
pushd ~/workspace
  git clone https://github.com/cloudfoundry-incubator/diego-release
  git clone https://github.com/cloudfoundry/cf-release
  git clone https://github.com/cloudfoundry-incubator/netman-release
popd

pushd ~/workspace/netman-release
  ./scripts/deploy-to-bosh-lite
popd
```

Then follow [the instructions for testing with the cats & dogs example](https://github.com/cloudfoundry-incubator/netman-release/tree/master/src/example-apps/cats-and-dogs).


## Deploy to AWS
0. Upload stemcell with Linux kernel 4.4 to bosh director
  - Current AWS stemcells only have 3.19
  - For now you will need a special stemcell that can be found [here](https://s3.amazonaws.com/dk-shared-assets/bosh-stemcell-3232.kernel-4.4-aws-xen-ubuntu-trusty-go_agent.tgz)
0. Create netman stubs
  - netman requires additional information in several stubs.
  - Add under `properties: uaa` in `stubs/cf/properties.yml`:

    ```
    scim:
      users:
      - admin|<admin-password>|scim.write,scim.read,openid,cloud_controller.admin,doppler.firehose
      - network-admin|<network-admin-password>|openid,network.admin
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
      netman-cf-acceptance:
        admin_password: <admin-password>
        admin_user: admin
        api: api.<system-domain>
        apps_domain: <apps-domain>
        nodes: 1
        skip_ssl_validation: true
        use_http: true
          test_user_password: <test-user-password>
      uaa:
        clients:
          network-policy:
            secret: <uaa-client-secret>
      policy-server:
        database_password: <db-password>
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
0. Run the acceptance errand
  ```
  bosh run errand netman-cf-acceptance
  ```

## To replace flannel with your own CNI plugin
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
0. Update the [deployment manifest properties](http://bosh.io/docs/deployment-manifest.html#properties)

  ```yaml
  garden-cni:
    adapter:
      cni_plugin_dir: /var/vcap/packages/YOUR_PACKAGE/bin # your CNI binary goes in this directory
      cni_config_dir: /var/vcap/jobs/YOUR_JOB/config/cni  # your CNI config file goes in this directory
  ```
  Remove any lingering references to `flannel` or `cni-flannel` in the deployment manifest.


## Development

### Running low-level tests

```bash
~/workspace/netman-release/scripts/docker-test
```

### Running the full acceptance test
WARNING: This test is taxing and has an aggressive timeout.
It may fail on a laptop or other underpowered bosh-lite.

```bash
bosh run errand netman-cf-acceptance
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
