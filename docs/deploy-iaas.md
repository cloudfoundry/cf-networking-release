# Deploy to a Cloud Infrastructure

We highly recommend using the new [CF Deployment](https://github.com/cloudfoundry/cf-deployment)
project as the basis for your deployment.

CF Deployment on its own does not include CF Networking, but the instructions
below will show you how to add it in.

## Using `cf-deployment`

This deployment option uses the new tooling:
- [bbl](https://github.com/cloudfoundry/bosh-bootloader), the bosh bootloader
- the new Golang [bosh-cli](https://github.com/cloudfoundry/bosh-cli)
- [cf-deployment](https://github.com/cloudfoundry/cf-deployment)

### Step 1: `bbl up` your BOSH and load balancers
Follow the instructions provided with [the `bbl` tool](https://github.com/cloudfoundry/bosh-bootloader)
to create a BOSH director and to create load balancers for CF.

The CF load balancer should use a certificate with a wildcard common name for your
Cloud Foundry.

After `bbl` is complete, it will create a `bbl-state.json` file.  Keep that in a secure, private directory
on your filesystem.

```bash
cd ~/my-deployment-credentials
ls
# bbl-state.json
```

Set some environment variables that will be used by subsequent commands:

```bash
export BOSH_USER=$(bbl director-username)
export BOSH_PASSWORD=$(bbl director-password)
export BOSH_ENVIRONMENT=$(bbl director-address)
export BOSH_CA_CERT=/tmp/$env-ca-cert
bbl director-ca-cert > $BOSH_CA_CERT
chmod 600 $BOSH_CA_CERT
export BOSH_DEPLOYMENT=cf
```

### Step 2: Seed your `vars-store`
If you don't have an existing deployment, create a new file which configures the
`system_domain` for the deployment.
```yaml
# vars-store.yml
system_domain: mysystem.example.com
```

The `system_domain` should match the common name of the cert you installed on your
`bbl` `cf` load-balancers.  For example if your cert was for `*.mycloudfoundry.mydomain`
then your `system_domain` should be `mycloudfoundry.mydomain`.

If you are upgrading an existing `cf-deployment`, use your existing `vars-store.yml`

### Step 3: Identify your "ops files"
Read the [instructions for CF Deployment Ops Files](https://github.com/cloudfoundry/cf-deployment#ops-files)
and identify the minimal set of ops files appropriate for your deployment.

For example, if you're deploying to AWS, you'll need (as of the time of this writing)
  - `$CF_DEPLOYMENT_REPO/operations/aws.yml`
  - `$CF_DEPLOYMENT_REPO/operations/change-logging-port-for-aws-elb.yml.yml`

In addition, **to use CF Networking, you'll need to include these two files**:

  - `$CF_NETWORKING_RELEASE_REPO/manifest-generation/opsfiles/cf-networking.yml`

  - `$CF_NETWORKING_RELEASE_REPO/manifest-generation/opsfiles/silk.yml`


### Step 4: BOSH deploy
Assemble and run a `bosh deploy` command that uses your ops files and `vars-store`:

For example, on GCP you might do:
```bash
bosh deploy \
  $CF_DEPLOYMENT_REPO/cf-deployment.yml \
  -o $CF_NETWORKING_RELEASE_REPO/manifest-generation/opsfiles/cf-networking.yml \
  -o $CF_NETWORKING_RELEASE_REPO/manifest-generation/opsfiles/silk.yml \
  --vars-store=vars-store.yml
```

For AWS, you might do:
```bash
bosh deploy \
  $CF_DEPLOYMENT_REPO/cf-deployment.yml \
  -o $CF_DEPLOYMENT_REPO/opsfiles/aws.yml \
  -o $CF_DEPLOYMENT_REPO/opsfiles/change-logging-port-for-aws-elb.yml \
  -o $CF_NETWORKING_RELEASE_REPO/manifest-generation/opsfiles/cf-networking.yml \
  -o $CF_NETWORKING_RELEASE_REPO/manifest-generation/opsfiles/silk.yml \
  --vars-store=vars-store.yml
```

Note that this command will add credentials (and other things) to your `vars-store.yml`.
Keep it in a private directory.

We keep ours next to the `bbl-state.json` file, checked into a private git repo.

### Step 5: Kick the tires

Get your credentials
```bash
export CF_API_ENDPONT=$(bosh int --path=/system_domain vars-store.yml)
export CF_ADMIN_USER=admin
export CF_ADMIN_PASSWORD=$(bosh int --path=/uaa_scim_users_admin_password vars-store.yml)
```

Get the [CF CLI](https://github.com/cloudfoundry/cli) if you don't already have it.

Log in to CF:
```bash
cf api $CF_API_ENDPOINT # if your cert was self-signed, you'll need to add --skip-ssl-validation
cf auth $CF_ADMIN_USER $CF_ADMIN_PASSWORD 
# or use 'cf login' for an interactive login
```

Then try out our [Cats and Dogs example](../src/example-apps/cats-and-dogs) on your new deployment.


---


## DEPRECATED: Using `cf-release` with `diego-release` tooling

This method is not recommended for new deployments.

This option requires the old Ruby bosh-cli, to be installed and aliased as `bosh`.

This deployment option assumes you already have a BOSH director on AWS where you have already successfully deployed Diego + Cloud Foundry,
by using the instructions and tooling in [the diego-release repo](https://github.com/cloudfoundry/diego-release/tree/develop/examples/aws).


1. Upload a recent stemcell (kernel 4.4 or higher) to the bosh director

2. Generate credentials

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

3. Edit the CF properties stub

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
