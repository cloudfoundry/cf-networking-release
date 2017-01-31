# Deploy to AWS

You have two options.  We recommend option #1 for new deployments.


## Option 1: Using `cf-deployment`

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
bosh-cli deploy \
  --vars-store=vars-store.yml \
  -o $CF_DEPLOYMENT_REPO/opsfiles/change-logging-port-for-aws-elb.yml \
  -o $CF_NETWORKING_RELEASE_REPO/manifest-generation/opsfiles/cf-networking.yml \
  $CF_DEPLOYMENT_REPO/cf-deployment.yml
```

Note that your `vars-store.yml` likely changed.  If you keep it in source control, commit.  But ensure it is in a private repository.  It holds credentials.

To kick the tires, try out our [Cats and Dogs example](../src/example-apps/cats-and-dogs) on your new deployment.


## Option 2: Using `cf-release` with `diego-release` tooling

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
    REPLACE_WITH_CLIENT_CERT
    REPLACE_WITH_CLIENT_KEY
    REPLACE_WITH_SERVER_CERT
    REPLACE_WITH_SERVER_KEY
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


0. Create a CF Networking stub `stubs/netman/stub.yml`:

    ```yaml
    ---
    netman_overrides:
      releases:
      - name: cf-networking
        version: latest
      driver_templates:
      - name: garden-cni
        release: cf-networking
      - name: cni-flannel
        release: cf-networking
      - name: netmon
        release: cf-networking
      - name: vxlan-policy-agent
        release: cf-networking
      properties:
        vxlan-policy-agent:
          policy_server_url: https://policy-server.service.cf.internal:4003
          ca_cert: |
            -----BEGIN CERTIFICATE-----
            REPLACE_WITH_CA_CERT
            -----END CERTIFICATE-----
          client_cert: |
            -----BEGIN CERTIFICATE-----
            REPLACE_WITH_CLIENT_CERT
            -----END CERTIFICATE-----
          client_key: |
            -----BEGIN RSA PRIVATE KEY-----
            REPLACE_WITH_CLIENT_KEY
            -----END RSA PRIVATE KEY-----
        policy-server:
          uaa_client_secret: REPLACE_WITH_UAA_CLIENT_SECRET
          skip_ssl_validation: true
          database:
            type: REPLACE_WITH_DB_TYPE # must be mysql or postgres
            username: REPLACE_WITH_USERNAME
            password: REPLACE_WITH_PASSWORD
            host: REPLACE_WITH_DB_HOSTNAME
            port: REPLACE_WITH_DB_PORT # e.g. 3306 for mysql
            name: REPLACE_WITH_DB_NAME # e.g. network_policy
          ca_cert: |
            -----BEGIN CERTIFICATE-----
            REPLACE_WITH_CA_CERT
            -----END CERTIFICATE-----
          server_cert: |
            -----BEGIN CERTIFICATE-----
            REPLACE_WITH_SERVER_CERT
            -----END CERTIFICATE-----
          server_key: |
            -----BEGIN RSA PRIVATE KEY-----
            REPLACE_WITH_SERVER_KEY
            -----END RSA PRIVATE KEY-----
        garden-cni:
          cni_plugin_dir: /var/vcap/packages/flannel/bin
          cni_config_dir: /var/vcap/jobs/cni-flannel/config/cni
        cni-flannel:
          flannel:
            etcd:
              require_ssl: (( config_from_cf.etcd.require_ssl))
          etcd_endpoints:
            - (( config_from_cf.etcd.advertise_urls_dns_suffix ))
          etcd_client_cert: (( config_from_cf.etcd.client_cert ))
          etcd_client_key: (( config_from_cf.etcd.client_key ))
          etcd_ca_cert: (( config_from_cf.etcd.ca_cert ))
      garden_properties:
        network_plugin: /var/vcap/packages/runc-cni/bin/garden-external-networker
        network_plugin_extra_args:
        - --configFile=/var/vcap/jobs/garden-cni/config/adapter.json
      jobs:
      - name: policy-server
        instances: 1
        persistent_disk: 256
        templates:
        - name: policy-server
          release: cf-networking
        - name: route_registrar
          release: cf
        - name: consul_agent
          release: cf
        - name: metron_agent
          release: cf
        resource_pool: database_z1
        networks:
          - name: diego1
        properties:
          nats:
            machines: (( config_from_cf.nats.machines ))
            user: (( config_from_cf.nats.user ))
            password: (( config_from_cf.nats.password ))
            port: (( config_from_cf.nats.port ))
          metron_agent:
            zone: z1
          route_registrar:
            routes:
            - name: policy-server
              port: 4002
              registration_interval: 20s
              uris:
              - (( "api." config_from_cf.system_domain "/networking" ))
          consul:
            agent:
              services:
                policy-server:
                  name: policy-server
                  check:
                    interval: 5s
                    script: /bin/true

    config_from_cf: (( merge ))
    ```

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
      -N ${environment_path}/stubs/netman/stub.yml \
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
