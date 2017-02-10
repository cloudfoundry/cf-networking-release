# Configuration Information for Operators

## Table of Contents
0. [Flannel Network Configuration](#flannel-network-configuration)
0. [Network Policy Database](#network-policy-database)
0. [MTU](#mtu)
0. [Mutual TLS](#mutual-tls)

## Flannel Network Configuration
The batteries-included connectivity solution uses [Flannel](https://github.com/coreos/flannel)
with the [VXLAN backend](https://github.com/coreos/flannel#backends).

The IP address allocation scheme is simple:

- The operator chooses a large contiguous address block for the
entire VXLAN network (`cf_networking.network`).
- The operator also chooses a uniform [subnet](https://en.wikipedia.org/wiki/Subnetwork) size (`cf_networking.subnet_size`).
- Flannel ensures that each Diego Cell (container host) is allocated a dedicated
a single subnet of that size from within that large block.
- The Flannel CNI plugin ensures that every container receives a unique IP within the subnet assigned to its host Cell.

In this way, every container in the installation receives a unique IP address.

#### BOSH properties
To configure the global network block and the size of the per-cell subnets, two
BOSH properties are used:

- `cf_networking.network`: The address block for the entire VXLAN network.
  Corresponds to the flannel configuration value `Network`.  Defaults to `10.255.0.0/16`

- `cf_networking.subnet_size`: The size, in bits, of the mask for the per-cell subnets.
  Corresponds to the flannel configuration value `SubnetLen`.  Defaults to `24`.

**Note**: the `cf_networking.network` property is consumed by two BOSH jobs: `garden-cni` and `cni-flannel`.  If you
intend to customize the network, you must set this property on both jobs.

**Note:** On BOSH-lite, avoid using or overlapping with the `10.244.0.0/16` or `10.254.0.0/16` ranges.
Those are both in use by BOSH-lite components and unpredictable behavior may result.

#### Network size limitations
The size of a given CF Networking installation is limited by the values of these two BOSH properties.

- let `s` be the value of `cf_networking.subnet_size`, e.g. `24` in the default case.
- let `n` be the prefix length in `cf_networking.network`, e.g. `16` in the default case.

Then:
- the number of containers on a given Diego cell cannot exceed `2^(32-s) - 2`
- the number of Diego cells in the installation cannot exceed `2^(s-n) - 1`
- the total number of containers running on the installation cannot exceed the product of the previous two numbers.

For example, using the default values, the maximum number of containers per cell is `2^(32-24) - 2 = 254`, 
the maximum number of cells in the installation is `2^(24-16) - 1 = 255`, and thus no more than `254 * 255 = 64770`
containers total may be running at a time on the installation.

Alternately, if `network` = `10.32.0.0/11` and `subnet_size` = `22` then the maximum number of
containers per cell would be `2^(32-22) - 2 = 1022`, the maximum number of cells
in the installation would be `2^(22-11) - 1 = 2047`, and no more than `1022 * 2047 = 2092034` containers
total may be running at a time on the installation.

**Note**: these upper bounds are for the network only.  Other limitations may also apply to your installation,
e.g. [`garden.max_containers`](https://github.com/cloudfoundry/garden-runc-release/blob/d67b61c/jobs/garden/spec#L106-L108).

#### Changing the network
It is safe to expand `cf_networking.network` on an existing deployment.
However it is not safe to modify `cf_networking.subnet_size`.  Unpredictable behavior may result.

Any changes which result in an IP range that does not completely contain the old network address block
must be done using
```
bosh deploy --recreate
```
and may cause the container network to become temporarily unavailable during the deploy.


## Network Policy Database
A SQL database is required to store Network Policies.  MySQL and PostgreSQL databases are currently supported.

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

- Add a logical database to the PostgreSQL instance included in cf-release,
  (which also holds the CloudController and UAA databases).  We use this in
  our [BOSH-lite setup](https://github.com/cloudfoundry-incubator/cf-networking-release/blob/d6ec307ba2/manifest-generation/cf-networking-bosh-lite-template.yml#L26-L35).

- BOSH-deploy the [Postgres release](https://github.com/cloudfoundry/postgres-release/)
  to a dedicated VM.

- Use a database service provided by your cloud infrastructure provider.  For example,
  in some of our automated tests we use an AWS RDS PostgreSQL instance configured as follows:

  - PostgreSQL 9.5.4
  - db.m3.medium (3.75 GiB)
  - 20 GB storage

### Policy database scale and performance testing
We have not done extensive performance testing of the network policy server and database.  However,
we have found performance to be acceptable with the above-mentioned CF-MySQL and RDS database
configurations when testing CF Networking features with:

  - 10 cells
  - 200 apps
  - 10 instances per app (i.e. 2000 app instances)
  - 4 policies per app (i.e. 800 policies)



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

- Flannel is a client of etcd

- The VXLAN Policy Agent is a client of the internal Policy Server API

Both of these connections require Mutual TLS.
(While etcd and flannel individually support unencrypted connections, we do not support this in CF Networking).

Scripts are available to generate certificate authorities, certificates and keys for these connections:

- [cf-release/scripts/generate-etcd-certs](https://github.com/cloudfoundry/cf-release/blob/master/scripts/generate-etcd-certs)
- [cf-networking-release/scripts/generate-certs](../scripts/generate-certs)

If you want to generate them yourself, ensure that the certificates for the
Policy Server and VXLAN Policy Agent support the cipher suite `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`.
The Policy Server will reject connections using any other cipher suite.

Below you will find detailed instructions for rotating these certificates with minimal downtime:

### Policy Server and Vxlan Policy Agent
To rotate your SSL certificates, keys, and certificate authorities, you must perform the following steps.

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


### Etcd and Flannel

To rotate your SSL certificates, keys, and certificate authorities, you must perform the following steps.

0. Generate new certificates by running [cf-release/scripts/generate-etcd-certs](https://github.com/cloudfoundry/cf-release/blob/master/scripts/generate-etcd-certs).

0. In your CF stub file, append the new CA certificate to the `properties.etcd.ca_cert` field.
  - **Do not remove the old CA certificate.**
  - Regenerate your CF and Diego manifests.
  - Deploy CF using your updated manifest.
  - Deploy Diego + CF Networking using your updated manifest.

  ```yaml
  ...
  properties:
    ...
    etcd:
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


0. In your CF stub file, replace the old server and client certificates and keys with the new certificates and keys.
  - **Do not remove the old CA certificate.**
  - Regenerate your CF and Diego manifests.
  - Deploy CF using your updated manifest.
  - Deploy Diego + CF Networking using your updated manifest.

  ```yaml
  ...
  properties:
    ...
    etcd:
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


0. In your CF stub file, remove the old CA certificate.
  - Regenerate your CF and Diego manifests.
  - Deploy CF using your updated manifest.
  - Deploy Diego + CF Networking using your updated manifest.

  ```yaml
  ...
  properties:
    ...
    etcd:
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
