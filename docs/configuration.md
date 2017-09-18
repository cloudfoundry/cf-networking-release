# Configuration Information for Operators

## Table of Contents
0. [Silk Network Configuration](#silk-network-configuration)
0. [Network Policy Access Control](#network-policy-access-control)
0. [Database Configuration](#database-configuration)
0. [MTU](#mtu)
0. [Mutual TLS](#mutual-tls)
0. [Max Open/Idle Connections](#max-openidle-connections)

## Silk Network Configuration
The default batteries-included connectivity solution uses [Silk](https://github.com/cloudfoundry-incubator/silk).
Previous versions used [Flannel](https://github.com/coreos/flannel), but that is no longer supported.

The IP address allocation scheme is simple:

- The operator chooses a large contiguous address block for the
entire VXLAN network (`cf_networking.network`).
- The operator also chooses a uniform [subnet](https://en.wikipedia.org/wiki/Subnetwork) size (`cf_networking.subnet_prefix_length`).
- Silk ensures that each Diego Cell (container host) is allocated a dedicated
a single subnet of that size from within that large block.
- The Silk CNI plugin ensures that every container receives a unique IP within the subnet assigned to its host Cell.

In this way, every container in the installation receives a unique IP address.

**Note**: by default, the `cf-deployment` will enable garden to use the Silk CNI plugin.
If you are not using `cf-deployment`, please add the following two properties to the `garden` job:
```
garden:
  network_plugin: /var/vcap/packages/runc-cni/bin/garden-external-networker
  network_plugin_extra_args:
  - --configFile=/var/vcap/jobs/garden-cni/config/adapter.json
```

#### BOSH properties
To configure the global network block and the size of the per-cell subnets, two
BOSH properties are used:

- `cf_networking.network`: The address block for the entire VXLAN network.
  Defaults to `10.255.0.0/16`

- `cf_networking.subnet_prefix_length`: The length, in bits, of the mask for the per-cell subnets.
  Must be less than 31 but larger than the prefix length for `network`.  Defaults to `24`.

**Note**: The `cf_networking.network` option should be configured to not overlap with anything on the infrastructure network used by BOSH, CF or services.
If the overlay network overlaps with anything on the underlay, traffic from the cell will not be able to reach that entity on the underlay.
To repair a deployment that has been misconfigured, follow our [recovery steps](troubleshooting.md#diagnosing-and-recovering-from-subnet-overlap)

**Note**: the `cf_networking.network` property is consumed by two BOSH jobs: `silk-daemon` and `silk-controller`.  If you
intend to customize the network, you must set this property on both jobs.

**Note:** On BOSH-lite, avoid using or overlapping with the `10.244.0.0/16` or `10.254.0.0/16` ranges.
Those are both in use by BOSH-lite components and unpredictable behavior may result.

#### Network size limitations
The size of a given CF Networking installation is limited by the values of these two BOSH properties.

- let `s` be the value of `cf_networking.subnet_prefix_length`, e.g. `24` in the default case.
- let `n` be the prefix length in `cf_networking.network`, e.g. `16` in the default case.

Then:
- the number of containers on a given Diego cell cannot exceed `2^(32-s) - 2`
- the number of Diego cells in the installation cannot exceed `2^(s-n) - 1`
- the total number of containers running on the installation cannot exceed the product of the previous two numbers.

For example, using the default values, the maximum number of containers per cell is `2^(32-24) - 2 = 254`,
the maximum number of cells in the installation is `2^(24-16) - 1 = 255`, and thus no more than `254 * 255 = 64770`
containers total may be running at a time on the installation.

Alternately, if `network` = `10.32.0.0/11` and `subnet_prefix_length` = `22` then the maximum number of
containers per cell would be `2^(32-22) - 2 = 1022`, the maximum number of cells
in the installation would be `2^(22-11) - 1 = 2047`, and no more than `1022 * 2047 = 2092034` containers
total may be running at a time on the installation.

**Note**: these upper bounds are for the network only.  Other limitations may also apply to your installation,
e.g. [`garden.max_containers`](https://github.com/cloudfoundry/garden-runc-release/blob/d67b61c/jobs/garden/spec#L106-L108).

#### Changing the network
It is safe to expand `cf_networking.network` on an existing deployment.
However it is not safe to modify `cf_networking.subnet_prefix_length`.  Unpredictable behavior may result.

Any changes which result in an IP range that does not completely contain the old network address block
must be done using
```
bosh deploy --recreate
```
and may cause the container network to become temporarily unavailable during the deploy.

## Network Policy Access Control

#### Network Admin Access
Any user with the `network.admin` UAA scope may create create network policies between any two applications.
There is no limit on the number of policies a network admin can configure.

#### App Developer Access
Application developers may be given a reduced set of permissions for configuring network policy.
In this permission model a user may configure policies between apps that are in spaces in which this user has the
`SpaceDeveloper` role in CloudController.  An application may be the source of only a limited number of
policies created this way (the limit is configurable via the BOSH property `cf_networking.max_policies_per_app_source`, defaults to 50).

- To grant an individual user this access, give them the `network.write` scope in UAA
- To grant **all** users this level of access, set the BOSH property `cf_networking.enable_space_developer_self_service` to `true`


## Database Configuration
A SQL database is required to store Subnet Leases and Network Policies.  MySQL and PostgreSQL databases are currently supported.

### Hosting options
The database may be hosted anywhere that the Policy Server BOSH VM can reach it,
including on another BOSH-deployed VM or on a cloud-provided service.  Here are some options:

#### MySQL

**Note:** The network policy database requires MySQL version of 5.7 or higher.

- Add a logical database to the CF-MySQL cluster that ships with
  [CF-Deployment](https://github.com/cloudfoundry/cf-deployment).  We've written
  a [CF-Networking opsfile](../manifest-generation/opsfiles/cf-networking.yml)
  to support this integration and we use it in some of our automated tests, configured
  as follows:

    - Single-node (not HA)
    - AWS m3.large VM
    - 10GB ephemeral disk
    - 10GB persistent disk

- BOSH-deploy the [CF-MySQL release](https://github.com/cloudfoundry/cf-mysql-release)
  to dedicated VM(s).  CF-MySQL may be deployed either as a single-node
  or as a highly available (HA) cluster.

- Use a database service provided by your cloud infrastructure provider.  For example,
  in some of our automated tests we use an AWS RDS MySQL instance configured as follows:

    - MySQL 5.7.16
    - db.t2.medium (4 Gib)
    - 20 GB storage


#### PostgreSQL

- Use a database service provided by your cloud infrastructure provider.  For example,
  in some of our automated tests we use an AWS RDS PostgreSQL instance configured as follows:

  - PostgreSQL 9.5.4
  - db.m3.medium (3.75 GiB)
  - 20 GB storage

- BOSH-deploy the [Postgres release](https://github.com/cloudfoundry/postgres-release/)
  to a dedicated VM.

### Policy Server DB scale and performance testing

Policy server performance has been validated for deployments with:

  - 100 cells
  - 20k applications
  - 1 instance per app
  - 60k policies
  - 20 requests per second

To reach these numbers we deployed:

  - 2 policy server instances (t2.large on AWS with 10GB ephemeral disk)
  - 1 CF MySQL instance (r3.4xlarge on AWS)

The bottleneck for performance seems to usually be the VM hosting the database.
If you are scaling above 30k policies, we suggest deploying the VM hosting the
database with a r3.4xlarge, a memory-intensive instance-type, if you are on
AWS.

We recommend having at least 2 instances of the policy server for high availability. We
saw little to no performance gain with 4 instances of the policy server for the
above scaling tests.

## MTU
Operators not using any additional encapsulation should not need to do any special configuration for MTUs.
The CNI plugins should automatically detect the host MTU and set the container MTU appropriately,
accounting for any overhead.

However, operators should understand that:
 - All Diego cells should be on the same network, and should have the same MTU
 - A change the Diego cell MTU will likely require the VMs to be recreated in
   order for the container network to function properly.

Operators using some additional encapsulation (e.g. ipsec) can manually configure the MTU for containers.
The configuration can be specified in the manifest under `properties.cf_networking.mtu`.
The operator should set the MTU low enough to account for the overhead of their own encapsulation plus the overhead from VXLAN.
As an example, if you are using ipsec with a recommended overhead of 100 bytes, and your VMs have MTU 1500,
you should set the MTU to 1350 (1500 - 100 for ipsec - 50 for VXLAN).


## Mutual TLS
In the batteries-included networking stack, there are two different control-plane connections between system components:

- The Silk Daemon is a client of the Silk Controller

- The VXLAN Policy Agent is a client of the internal Policy Server API

Both of these connections require Mutual TLS.

A script is available to generate certificate authorities, certificates and keys for these connections:

- [cf-networking-release/scripts/generate-certs](../scripts/generate-certs)

If you want to generate them yourself, ensure that all certificates
support the cipher suite `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`.
The Policy Server and Silk Controller will reject connections using any other cipher suite.

Below you will find detailed instructions for rotating these certificates with minimal downtime:


### Rotating Certs

To rotate your SSL certificates, keys, and certificate authorities, you must perform the following steps.

**Note:** The example below shows the steps for rotating the certs on the `policy-server` and `vxlan-policy-agent`,
but the same procedure can be used to rotate the certs for the `silk-daemon` and `silk-controller`.

  0. Generate new certificates by running `./scripts/generate-certs`.

  0. In your CF Networking stub file, append the new CA certificate (contents of `cf-networking-certs/cf-networking-ca.crt`)
  to the `cf_networking_overrides.properties.cf_networking.policy_server.ca_cert`
  and `cf_networking_overrides.properties.cf_networking.vxlan_policy_agent.ca_cert` fields.
      - **Do not remove the old CA certificates.**
      - Regenerate your Diego manifest.
      - Deploy Diego + CF Networking using your updated manifest.

      ```yaml
      ---
      cf_networking_overrides:
        ...
        properties:
          ...
          cf_networking:
            ...
            policy_server:
              ...
              ca_cert: |
                -----BEGIN CERTIFICATE-----
                ###########################################################
                #######           Your Old CA Certificate           #######
                ###########################################################
                -----END CERTIFICATE-----
                -----BEGIN CERTIFICATE-----
                ###########################################################
                #######           Your New CA Certificate           #######
                ###########################################################
                -----END CERTIFICATE-----
              server_cert: |
                -----BEGIN CERTIFICATE-----
                ###########################################################
                #######           Your Old Server Certificate       #######
                ###########################################################
                -----END CERTIFICATE-----
              server_key: |
                -----BEGIN RSA PRIVATE KEY-----
                ###########################################################
                #######           Your Old Server Key                ######
                ###########################################################
                -----END RSA PRIVATE KEY-----
            vxlan_policy_agent:
              ...
              ca_cert: |
                -----BEGIN CERTIFICATE-----
                ###########################################################
                #######           Your Old CA Certificate           #######
                ###########################################################
                -----END CERTIFICATE-----
                -----BEGIN CERTIFICATE-----
                ###########################################################
                #######           Your New CA Certificate           #######
                ###########################################################
                -----END CERTIFICATE-----
              client_cert: |
                -----BEGIN CERTIFICATE-----
                ###########################################################
                #######           Your Old Client Certificate       #######
                ###########################################################
                -----END CERTIFICATE-----
              client_key: |
                -----BEGIN RSA PRIVATE KEY-----
                ###########################################################
                #######           Your Old Client Key                ######
                ###########################################################
                -----END RSA PRIVATE KEY-----
            ...
      ```


  0. In your CF Networking stub file, replace the old server and client certificates and keys with the new certificates and keys.
      - **Do not remove the old CA certificates.**
      - Regenerate your Diego manifest.
      - Deploy Diego + CF Networking using your updated manifest.

      ```yaml
      ---
      cf_networking_overrides:
        ...
        properties:
          ...
          cf_networking:
            ...
            policy_server:
              ...
              ca_cert: |
                -----BEGIN CERTIFICATE-----
                ###########################################################
                #######           Your Old CA Certificate           #######
                ###########################################################
                -----END CERTIFICATE-----
                -----BEGIN CERTIFICATE-----
                ###########################################################
                #######           Your New CA Certificate           #######
                ###########################################################
                -----END CERTIFICATE-----
              server_cert: |
                -----BEGIN CERTIFICATE-----
                ###########################################################
                #######           Your New Server Certificate       #######
                ###########################################################
                -----END CERTIFICATE-----
              server_key: |
                -----BEGIN RSA PRIVATE KEY-----
                ###########################################################
                #######           Your New Server Key                ######
                ###########################################################
                -----END RSA PRIVATE KEY-----
            vxlan_policy_agent:
              ...
              ca_cert: |
                -----BEGIN CERTIFICATE-----
                ###########################################################
                #######           Your Old CA Certificate           #######
                ###########################################################
                -----END CERTIFICATE-----
                -----BEGIN CERTIFICATE-----
                ###########################################################
                #######           Your New CA Certificate           #######
                ###########################################################
                -----END CERTIFICATE-----
              client_cert: |
                -----BEGIN CERTIFICATE-----
                ###########################################################
                #######           Your New Client Certificate       #######
                ###########################################################
                -----END CERTIFICATE-----
              client_key: |
                -----BEGIN RSA PRIVATE KEY-----
                ###########################################################
                #######           Your New Client Key                ######
                ###########################################################
                -----END RSA PRIVATE KEY-----
            ...
      ```


  0. In your CF Networking stub file, remove the old CA certificate from the `cf_networking_overrides.properties.cf_networking.policy_server.ca_cert`
  and `cf_networking_overrides.properties.cf_networking.vxlan_policy_agent.ca_cert` fields.
      - Regenerate your Diego manifest.
      - Deploy Diego + CF Networking using your updated manifest.

      ```yaml
      ---
      cf_networking_overrides:
        ...
        properties:
          ...
          cf_networking:
            ...
            policy_server:
              ...
              ca_cert: |
                -----BEGIN CERTIFICATE-----
                ###########################################################
                #######           Your New CA Certificate           #######
                ###########################################################
                -----END CERTIFICATE-----
              server_cert: |
                -----BEGIN CERTIFICATE-----
                ###########################################################
                #######           Your New Server Certificate       #######
                ###########################################################
                -----END CERTIFICATE-----
              server_key: |
                -----BEGIN RSA PRIVATE KEY-----
                ###########################################################
                #######           Your New Server Key                ######
                ###########################################################
                -----END RSA PRIVATE KEY-----
            vxlan_policy_agent:
              ...
              ca_cert: |
                -----BEGIN CERTIFICATE-----
                ###########################################################
                #######           Your New CA Certificate           #######
                ###########################################################
                -----END CERTIFICATE-----
              client_cert: |
                -----BEGIN CERTIFICATE-----
                ###########################################################
                #######           Your New Client Certificate       #######
                ###########################################################
                -----END CERTIFICATE-----
              client_key: |
                -----BEGIN RSA PRIVATE KEY-----
                ###########################################################
                #######           Your New Client Key                ######
                ###########################################################
                -----END RSA PRIVATE KEY-----
            ...
      ```

## Max Open/Idle Connections

In order to limit the number of open or idle connections between the silk daemon and silk controller, the following properties can be set.
- `cf_networking.silk_controller.max_open_connections`
- `cf_networking.silk_controller.max_idle_connections`

By default there is no limit to the number of open or idle connections.
