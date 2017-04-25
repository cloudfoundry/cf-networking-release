## Manifest property changes

See [AWS](iaas.md#deploy-to-aws) deployment docs for examples

### 0.22.0

**Changed Properties**

  - `cf_networking.vxlan_policy_agent.iptables_c2c_logging` is now just `cf_networking.iptables_c2c_logging`
     It can be set on both the vxlan-policy-agent job for accepted packets 
     as well as the connectivity-plugin job for denied packets and defaults to false on both.

**Removed Properties**

  - `cf_networking.vxlan_policy_agent.subnet_file` is now removed.
  - `cf_networking.network` has been removed from the `garden-cni` job spec.  The property remains in use on other jobs.

### 0.21.0

**Changed Properties**

  - The value for `cf_networking.garden_external_networker.cni_plugin_dir` now defaults to `/var/vcap/packages/silk-cni/bin`
    We recommend that you remove any overrides for this property, unless you are intending to use a 3rd party CNI plugin.

### 0.20.0

**Changed Properties**

  - The value for `cf_networking.garden_external_networker.cni_plugin_dir` now defaults to `/var/vcap/packages/silk/bin`
    We recommend that you remove any overrides for this property, unless you are intending to use a 3rd party CNI plugin.

### 0.19.0

**Changed Properties**

  - The value for `cf_networking.garden_external_networker.cni_plugin_dir` **must** be updated to `/var/vcap/packages/silk/bin`
    if you are not swapping out CNI with your own plugin. (There is no default currently, but we plan to add one in the next release)
  - The property for global ASG logging has changed from `cf_networking.garden_external_networker.iptables_asg_logging`
    to `cf_networking.iptables_asg_logging`.

**Removed Properties**

 - `cf_networking.flannel_watchdog.no_bridge` is now removed.

**New Properties**

A new property `dns_servers` has been added to enable upcoming BOSH DNS support for app containers.
The servers (specified as a list of strings) will be used to populate the `/etc/resolv.conf` file in
the container.  To use this feature, operators must be using garden-runc-release version >= 1.4.0.
Set

- `cni-flannel` job, with property `cf_networking.dns_servers`

For example:
```yaml
cf_networking:
   dns_servers:
      - 169.254.0.2
```

If a link-local address is specified (as in the example above), the iptables on the host will
be modified to allow the container to access that address.

If this property is not set (or left with its default value of `[]`) then Garden-runC will set the list
based on its own BOSH properties.  By default, the DNS servers from the host are used.


### 0.18.0

**New Properties**

  - `cf_networking.rep_listen_addr_admin` enables our drain scripts to wait for the Diego rep to exit.
  It should always be the same value as `diego.rep.listen_addr_admin`. It defaults to `127.0.0.1:1800`.
  - `cf_networking.garden_external_networker.iptables_asg_logging` globally enables iptables logging for
    all ASGs, including logging of denied packets. Defaults to false.
  - `cf_networking.vxlan_policy_agent.iptables_c2c_logging` enables iptables logging for
  container-to-container traffic.  It defaults to `false`. *Note: this is already
  [configurable at runtime](troubleshooting.md#enabling-iptables-logging-for-container-to-container-traffic).*
  - `cf_networking.plugin.health_check_port` allows BOSH to better health-check the `flanneld` process
  required for connectivity.

**Removed Properties**

 - `cf_networking.policy_server.database.connection_string` was deprecated in v0.10.0 and is now removed.

### 0.17.0
Policy server requires a CA cert for UAA, **manifest must be generated with `diego-release` v1.7.0+**

The following needs to be added to your `cf-networking` stub **even if you are skipping ssl validation of UAA**:

```diff
cf_networking_overrides:
  properties:
    cf_networking:
      policy_server:
+      uaa_ca: (( config_from_cf.uaa.ca_cert ))
```

### 0.15.0

**Many breaking changes!**

0. Requires Diego Release v1.6.2 or higher

0. Release name changed from `netman` to `cf-networking`

0. Acceptance errand name changed from `netman-cf-acceptance` to `cf-networking-acceptance`

0. All propeties of `cf-networking-release` jobs live under a global `properties.cf_networking` key e.g:

  ```diff
     properties:
  +    cf_networking:
  +      ...
  ```

0. Several references to jobs in properties have changed names:

  ```diff
     properties:
  +    cf_networking:
  -      policy-server:
  +      policy_server:
           ...
  -      vxlan-policy-agent:
  +      vxlan_policy_agent:
           ...
  -      cni-flannel:
  +      plugin:
           ...
  -      garden-cni:
  +      garden_external_networker:
           ...
  +      ...
  ```

0. `properties.netman.disable` renamed `properties.cf_networking.disable`

0. `flannel.etcd.require_ssl` property has been completely removed.
Previously it defaulted to `true` but could be overridden to `false`.
Now SSL is required for the flannel connection to etcd.
Refer to the [manifest generation docs](https://github.com/cloudfoundry-incubator/cf-networking-release/blob/develop/docs/iaas.md#deploy-to-aws)
for details on how to generate and configure certs and keys.
Note, you will likely need to make similar changes to other etcd clients, e.g. loggregator.

0. In the stub file, `netman_overrides` renamed to `cf_networking_overrides`

### 0.12.0

In the CF properties stub:

```diff
    -   scope: cloud_controller.read,cloud_controller.write,openid,password.write,cloud_controller.admin,scim.read,scim.write,doppler.firehose,uaa.user,routing.router_groups.read,network.admin
    +   scope: cloud_controller.read,cloud_controller.write,openid,password.write,cloud_controller.admin,scim.read,scim.write,doppler.firehose,uaa.user,routing.router_groups.read,network.admin,network.write
```

###  0.10.0
Policy Server database connection is now expressed as a set of config options, not a single connection string

In the CF Networking stub:

```diff
  policy_server:
    database:
       type: REPLACE_WITH_DB_TYPE # must be mysql or postgres
-      connection_string: postgres://USERNAME:PASSWORD@DB_HOSTNAME:5524/DB_NAME?sslmode=disable
+      username: REPLACE_WITH_USERNAME
+      password: REPLACE_WITH_PASSWORD
+      host: REPLACE_WITH_DB_HOSTNAME
+      port: REPLACE_WITH_DB_PORT # e.g. 3306 for mysql
+      name: REPLACE_WITH_DB_NAME # e.g. network_policy
```

###  0.7.0

CF Networking stub

```diff
        policy-server:
          uaa_client_secret: REPLACE_WITH_UAA_CLIENT_SECRET
          uaa_url: (( "https://uaa." config_from_cf.system_domain ))
+         cc_url: (( "https://api." config_from_cf.system_domain ))
          skip_ssl_validation: true
```

CF stub

```diff
     network-policy:
-      authorities: uaa.resource
+      authorities: uaa.resource,cloud_controller.admin_read_only
+      authorized-grant-types: client_credentials,refresh_token
       secret: REPLACE_WITH_UAA_CLIENT_SECRET
```
