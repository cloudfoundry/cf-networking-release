# Configuration Information for Operators

## Table of Contents
0. [Flannel Network Configuration](#flannel-network-configuration)
0. [Network Policy Database](#network-policy-database)
0. [MTU](#mtu)
0. [Mutual TLS](#mutual-tls)
0. [SSL Certificate, Key, and Certificate Authority Rotation](#ssl-certificate-key-and-certificate-authority-rotation)

### Flannel Network Configuration
The default flannel network is `10.255.0.0/16` which will allow for a maximum of 256 cells.

The network can be configured via the bosh property `cf_networking.network`
which is used by both the `cni-flannel` and `garden-cni` jobs.

For instance, to allow for up to 4096 cells, `10.240.0.0/12` could be used.

It is safe to expand the network on a subsequent deploy.

However, any changes which result in an IP range that does not completely contain the current network
must be done with the --recreate option and may result in containers being unable to reach each
other during the deploy.

**Note:** On bosh-lite, the network should avoid any IP addresses that include the
10.244 or 10.254 ranges, as those are both in use by bosh components.


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
