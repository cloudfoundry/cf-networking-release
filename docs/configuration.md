# Configuration Information for Operators

## Table of Contents
0. [Flannel Network Configuration](#flannel-network-configuration)
0. [Network Policy Database](#network-policy-database)
0. [MTU](#mtu)
0. [Mutual TLS](#mutual-tls)
0. [SSL Certificate, Key, and Certificate Authority Rotation](#ssl-certificate-key-and-certificate-authority-rotation)

### Flannel Network IP Address Management
The batteries-included connectivity solution uses [Flannel](https://github.com/coreos/flannel)
with the [VXLAN backend](https://github.com/coreos/flannel#backends).

IP address allocation scheme is simple: the operator chooses a large contiguous address block for the
entire VXLAN network (`cf_networking.network`).  The operator also chooses a uniform subnet size (`cf_networking.subnet_size`).
Flannel ensures that each Diego Cell (container host) is allocated a dedicated
a single [subnet](https://en.wikipedia.org/wiki/Subnetwork) of that size from within that large block.  And the Flannel CNI plugin
ensures that every container receives a unique IP within its cell's subnet.

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
Taken together, the two BOSH properties define upper bounds on the size of a given CF Networking installation.

- let `s` be the value of `cf_networking.subnet_size`, e.g. `24` in the default case.
- let `n` be the subnet mask length in `cf_networking.network`, e.g. `16` in the default case.

Then:
- the number of containers on a given Diego cell cannot exceed: `2^(32-s) - 2`
- the number of Diego cells in the installation cannot exceed: `2^(s-n) - 1`

For example, using the default values, the maximum number of containers per cell is `2^(32-24) - 2 = 254`
and the maximum number of cells in the installation is `2^(24-16) - 1 = 255`

Alternately, if `network` = `10.32.0.0/11` and `subnet_size` = `22` then the maximum number of
containers per cell would be `2^(32-22) - 2 = 1022` and the maximum number of cells
in the installation would be `2^(22-11) - 1 = 2047`.

**Note**: these upper bounds are for the network only.  Other limitations may also apply to your installation,
e.g. [`garden.max_containers`](https://github.com/cloudfoundry/garden-runc-release/blob/d67b61c/jobs/garden/spec#L106-L108).

#### Changing the network
It is safe to expand `cf_networking.network` on an existing installation.  It is not safe to modify `cf_networking.subnet_size`
on an existing deployment.

However, any changes which result in an IP range that does not completely contain the old network address block
must be done with the --recreate option and may cause the container network to become temporarily unavailable during the deploy.


### Network Policy Database
Both the MySQL and PostgreSQL dialects of SQL are supported on CF Networking.

Operators have a choice for deployment styles for both MySQL and PostgreSQL data stores.

**Note:** The current scale of performance testing we have done with the
RDS instance configurations shown below is:
  - 10 cells
  - 200 apps
  - 10 instances per app (i.e. 2000 app instances)
  - 4 policies per app (i.e. 800 policies)


#### MySQL
For MySQL, operators have at least the following options:
  - Use the [CF-MySQL release](https://github.com/cloudfoundry/cf-mysql-release)
  in standalone mode as a separate BOSH deployment,
  either as a single node, or as a highly available (HA) cluster.
  - Use an infrastructure-specific database deployment, such as an RDS MySQL
  instance on AWS.
  - Add a database to the MySQL cluster that comes with
  [CF-Deployment](https://github.com/cloudfoundry/cf-deployment). There
  is a [CF-Networking opsfile](../manifest-generation/opsfiles/cf-networking.yml)
  that adds `network_policy` to the `seeded_databases`.

For testing, we have an AWS RDS MySQL instance with these properties:
  - Version - Dev/Test
  - Engine - MySQL 5.7.16
  - DB Instance Class - db.t2.medium (4 Gib)
  - Storage - 20 GB

**Note:** The network policy database requires a MySQL version of 5.7 or higher.


#### PostgreSQL
For PostgreSQL, operators have at least the following options:
  - Use the PostgreSQL job from the CF release, either sharing an existing instance
  that houses the CC and UAA databases, or deploying a separate node specifically
  for the network policy database.
  - Use an infrastructure-specific database deployment, such as an RDS PostgreSQL
  instance on AWS.

For testing, we have two AWS RDS PostgreSQL instances with these properties:
  - Version - Dev/Test
  - Engine - PostgreSQL 9.5.4 and PostgreSQL 9.6.1
  - DB Instance Class - db.m3.medium (3.75 GiB) and db.t2.medium (4 Gib)
  - Storage - 20 GB and 20 GB

**Note:** The network policy database requires a MySQL version of 5.7 or higher.


### MTU
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


### Mutual TLS
The policy server exposes its internal API over mutual TLS.  We provide [a script](../scripts/generate-certs)
to generate these certificates for you.  If you want to generate them yourself,
ensure that the certificates support the cipher suite `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`.
The Policy Server will reject connections using any other cipher suite.


### SSL Certificate, Key, and Certificate Authority Rotation

#### Policy Server and Vxlan Policy Agent
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


#### Etcd and Flannel

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
