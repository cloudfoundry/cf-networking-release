## Deploy to AWS
0. Upload stemcell with Linux kernel 4.4 to bosh director.  Versions >= 3263.2 should work.
0. Create netman stubs

  - Add under `properties: uaa` in `stubs/cf/properties.yml`:

    ```yaml
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
          - network.admin
    clients:
      cf:
        scope: cloud_controller.read,cloud_controller.write,openid,password.write,cloud_controller.admin,scim.read,scim.write,doppler.firehose,uaa.user,routing.router_groups.read,network.admin
      network-policy:
        authorities: uaa.resource
        secret: <network-policy-secret>
    ```


  - Create a netman stub `stubs/netman/stub.yml`:

    - The policy-agent communicates with the policy-server using mutual TLS.
      Generate PEM encoded certs and keys for `vxlan-policy-agent` and `policy-server` and update the associated properties.
        - See the [generate-certs](scripts/generate-certs.sh) script for an example
    - All other fields with `REPLACE_*` values must be provided

    ```yaml
    ---
    netman_overrides:
      releases:
      - name: netman
        version: latest
      driver_templates:
      - name: garden-cni
        release: netman
      - name: cni-flannel
        release: netman
      - name: netmon
        release: netman
      - name: vxlan-policy-agent
        release: netman
      properties:
        vxlan-policy-agent:
          policy_server_url: https://policy-server.service.cf.internal:4003
          ca_cert: REPLACE_WITH_SERVER_CA_CERT
          client_cert: REPLACE_WITH_CLIENT_CERT
          client_key: REPLACE_WITH_CLIENT_KEY
        policy-server:
          uaa_client_secret: REPLACE_WITH_UAA_CLIENT_SECRET
          uaa_url: (( "https://uaa." config_from_cf.system_domain ))
          skip_ssl_validation: true
          database:
            type: REPLACE_WITH_DB_TYPE # mysql or postgres
            connection_string: REPLACE_WITH_DB_CONNECTION_STRING
          ca_cert: REPLACE_WITH_CLIENT_CA_CERT
          server_cert: REPLACE_WITH_SERVER_CERT
          server_key: REPLACE_WITH_SERVER_KEY
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
          release: netman
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

0. Generate diego with netman manifest:
  - Run the following bash script. Set `environment_path` to the directory containing your stubs for cf, diego, and netman.
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

