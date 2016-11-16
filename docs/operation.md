# Information for operators

### MTU
Operators should not need to do any special configuration for MTUs.  The CNI plugins
should automatically detect the host MTU and set the container MTU appropriately, 
accounting for any overhead.

However, operators should understand that:
 - All Diego cells should be on the same network, and should have the same MTU
 - A change the Diego cell MTU will likely require the VMs to be recreated in
   order for the container network to function properly.

### Mutual TLS
The policy server exposes its internal API over mutual TLS.  We provide [a script](../scripts/generate-certs)
to generate these certificates for you.  If you want to generate them yourself,
ensure that the certificates support the cipher suite `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`.
The Policy Server will reject connections using any other cipher suite.

### SSL Certificate, Key, and Certificate Authority Rotation

#### Policy Server and Vxlan Policy Agent

To rotate your SSL certificates, keys, and certificate authorities, you must perform the following steps.

0. Generate new certificates by running `./scripts/generate-certs`.

0. In your netman stub file, append the new CA certificate (contents of `netman-certs/netman-ca.crt`)
to the `netman_overrides.properties.policy-server.ca_cert`
and `netman_overrides.properties.vxlan-policy-agent.ca_cert` fields.
  - **Do not remove the old CA certificates.**
  - Regenerate your Diego manifest.
  - Deploy Diego + Netman using your updated manifest.

  ```yaml
  ---
  netman_overrides:
    ...
    properties:
      ...
      policy-server:
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
      vxlan-policy-agent:
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


0. In your netman stub file, replace the old server and client certificates and keys with the new certificates and keys. 
  - **Do not remove the old CA certificates.**
  - Regenerate your Diego manifest.
  - Deploy Diego + Netman using your updated manifest.

  ```yaml
  ---
  netman_overrides:
    ...
    properties:
      ...
      policy-server:
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
      vxlan-policy-agent:
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


0. In your netman stub file, remove the old CA certificate from the `netman_overrides.properties.policy-server.ca_cert`
and `netman_overrides.properties.vxlan-policy-agent.ca_cert` fields.
  - Regenerate your Diego manifest.
  - Deploy Diego + Netman using your updated manifest.

  ```yaml
  ---
  netman_overrides:
    ...
    properties:
      ...
      policy-server:
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
      vxlan-policy-agent:
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
  - Deploy Diego + Netman using your updated manifest.

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
  - Deploy Diego + Netman using your updated manifest.

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
  - Deploy Diego + Netman using your updated manifest.

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
