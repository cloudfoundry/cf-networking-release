# Deploy to GCP

Follow the instructions for deploying to AWS with some differences:
 - You have a BOSH director on GCP that was created using [bbl](https://github.com/cloudfoundry/bosh-bootloader).
 - When you `bosh-cli deploy` the CF Deployment opsfile you will use is: `$CF_DEPLOYMENT_REPO/opsfiles/gcp.yml`.


# Deploy to AWS

You have two options.  We recommend option #1 for new deployments.

## Using `cf-deployment`

This deployment option uses the new tooling:
- [bbl](https://github.com/cloudfoundry/bosh-bootloader), the bosh bootloader
- the new Golang [bosh-cli](https://github.com/cloudfoundry/bosh-cli)
- [cf-deployment](https://github.com/cloudfoundry/cf-deployment), refer to our [release notes](https://github.com/cloudfoundry-incubator/cf-networking-release/releases) to get information on validated versions

It assumes you have a BOSH director on AWS that was created using [the `bbl` tool](https://github.com/cloudfoundry/bosh-bootloader).

You'll need `bbl` installed on your local machine in order to acquire the credentials for the bosh director.  Grab [the latest release from GitHub](https://github.com/cloudfoundry/bosh-bootloader/releases).

You should have a private directory in which you hold the `bbl-state.json` file for your bosh director

```bash
cd ~/my-deployment-credentials
ls
# bbl-state.json
```

```bash
export BOSH_USER=$(bbl director-username)
export BOSH_PASSWORD=$(bbl director-password)
export BOSH_ENVIRONMENT=$(bbl director-address)
export BOSH_CA_CERT=/tmp/$env-ca-cert
bbl director-ca-cert > $BOSH_CA_CERT
chmod 600 $BOSH_CA_CERT
export BOSH_DEPLOYMENT=cf
```

If you are upgrading an existing `cf-deployment`, this same directory should hold your `vars-store.yml`
file containing credentials for your existing deployment.

If you don't have an existing deployment, you can seed one with the following contents:
```yaml
# vars-store.yml
system_domain: mysystem.example.com
```

Then deploy
```bash
bosh deploy \
  $CF_DEPLOYMENT_REPO/cf-deployment.yml \
  -o $CF_DEPLOYMENT_REPO/opsfiles/change-logging-port-for-aws-elb.yml \
  -o $CF_NETWORKING_RELEASE_REPO/manifest-generation/opsfiles/cf-networking.yml \
  -o $CF_NETWORKING_RELEASE_REPO/manifest-generation/opsfiles/silk.yml \
  --vars-store=vars-store.yml
```

Note that your `vars-store.yml` likely changed.  If you keep it in source control, commit.  But ensure it is in a private repository.  It holds credentials.

To kick the tires, try out our [Cats and Dogs example](../src/example-apps/cats-and-dogs) on your new deployment.


## DEPRECATED: Using `cf-release` with `diego-release` tooling

Note: Using this option requires the old Ruby bosh-cli to be installed and aliased as `bosh`.

This deployment option assumes you already have a BOSH director on AWS where you have already successfully deployed Diego + Cloud Foundry,
by using the instructions and tooling in [the diego-release repo](https://github.com/cloudfoundry/diego-release/tree/develop/examples/aws).

0. Upload stemcell with Linux kernel 4.4 to bosh director.  Versions >= 3263.2 should work.

0. Generate credentials
  - Create a strong password for a new UAA client to be called `network-policy`.  We'll refer to this
    with the string `REPLACE_WITH_UAA_CLIENT_SECRET` below.
  - Generate certs & keys for mutual TLS between the policy server and policy agents.  You can use our
    [handy script](../scripts/generate-certs) to create these.  We'll refer to these with the strings

    ```
    REPLACE_WITH_CA_CERT
    REPLACE_WITH_POLICY_AGENT_CERT
    REPLACE_WITH_POLICY_AGENT_KEY
    REPLACE_WITH_POLICY_SERVER_CERT
    REPLACE_WITH_POLICY_SERVER_KEY
    REPLACE_WITH_SILK_DAEMON_CERT
    REPLACE_WITH_SILK_DAEMON_KEY
    REPLACE_WITH_SILK_CONTROLLER_CERT
    REPLACE_WITH_SILK_CONTROLLER_KEY
    ```

0. Edit the CF properties stub

  - Add under `properties.uaa.scim.users` the group `network.admin` for `admin`
    ```diff
    scim:
      users:
      - name: admin
        password: <admin-password>
        groups:
          - scim.write
          - scim.read
          - openid
          - cloud_controller.admin
          - clients.read
          - clients.write
          - doppler.firehose
          - routing.router_groups.read
          - routing.router_groups.write
    +     - network.admin
    ```

  - Add under `properties.uaa.clients`

    ```diff
    clients:
      cf:
    -   scope: cloud_controller.read,cloud_controller.write,openid,password.write,cloud_controller.admin,scim.read,scim.write,doppler.firehose,uaa.user,routing.router_groups.read
    +   scope: cloud_controller.read,cloud_controller.write,openid,password.write,cloud_controller.admin,scim.read,scim.write,doppler.firehose,uaa.user,routing.router_groups.read,network.admin,network.write
    + network-policy:
    +   authorities: uaa.resource,cloud_controller.admin_read_only
    +   authorized-grant-types: client_credentials,refresh_token
    +   secret: REPLACE_WITH_UAA_CLIENT_SECRET
    ```


0. Copy the [example IaaS stub file](../manifest-generation/stubs/example-iaas-diego-cf-networking.yml)
   to `stubs/cf-networking/stub.yml` and fill in values as needed.

0. Generate Diego with CF Networking manifest:
  - Run the following bash script. Set `environment_path` to the directory containing your stubs for CF, Diego, and CF Networking.
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
      -N ${environment_path}/stubs/cf-networking/stub.yml \
      -v ${environment_path}/stubs/diego/release-versions.yml \
      > ${output_path}/diego.yml
  popd
  ```

0. Deploy
  - Target your bosh director.
  ```bash
  bosh target <your-director>
  ```
  - Set the deployment
  ```bash
  bosh deployment ${output_path}/diego.yml
  ```
  - Deploy
  ```bash
  bosh deploy
  ```


0. Kicking the tires

   Try out our [Cats and Dogs example](../src/example-apps/cats-and-dogs) on your new deployment.
