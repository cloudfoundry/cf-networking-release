## Manifest changelog

See [deployment docs](https://github.com/cloudfoundry/cf-deployment) for examples

### 2.18.0 CF-Networking-Release
**New Properties**
  - An optional parameter has been added to the `bosh-dns-adapter` job to allow for internal service mesh domains. Routes created with these domains will be proxied through the sidecar envoy. This is a part of istio integration. Defaults to `[]`
    - `internal_service_mesh_domains`
  - An optional parameter has been added to the `policy-server` job to skip host name validation when using ssl validation. The `policy-server-internal` uses the same configuration applied to `policy-server` via bosh links.
    - `database.skip_hostname_validation`

### 2.18.0 Silk-Release
**New Properties**
  - An optional parameter has been added to the `silk-controller` job to skip host name validation when using ssl validation
    - `database.skip_hostname_validation`

### 2.17.0 CF-Networking-Release
**New Properties**
  - An optional parameter has been added to the `policy-server-internal` job to enforce dynamic egress policy. Note that you can still create dynamic egress policies through the external API. Defaults to `false`
    - `enforce_experimental_dynamic_egress_policies`

### 2.17.0 Silk-Release
**No Manifest Changes**

### 2.16.0 CF-Networking-Release
**New Properties**
  - An optional parameter has been added to the `policy-server` and `policy-server-internal` jobs to configure the max lifetime of connections to the policy-server database.
    - `connections_max_lifetime_seconds`

### 2.16.0 Silk-Release
**New Properties**
  - An optional parameter has been added to the `silk-controller` job to configure the max lifetime of connections to the silk database.
    - `connections_max_lifetime_seconds`

### 2.15.0 CF-Networking-Release & Silk-Release
**No Manifest Changes**

### 2.14.0 CF-Networking-Release & Silk-Release
**No Manifest Changes**

### 2.13.0 CF-Networking-Release
**New Default**
  - The `internal_domains` property on the `bosh-dns-adapter` job now defaults to `[]` instead of `[apps.internal.]`  

### 2.13.0 Silk-Release
**No Manifest Changes**

### 2.12.0 CF-Networking-Release & Silk-Release
**No Manifest Changes**

### 2.11.0 CF-Networking-Release & Silk-Release
**No Manifest Changes**

### 2.10.0 CF-Networking-Release & Silk-Release
**No Manifest Changes**

### 2.9.0 CF-Networking-Release & Silk-Release
**No Manifest Changes**

### 2.8.0 CF-Networking-Release & Silk-Release
**No Manifest Changes**

### 2.7.0 CF-Networking-Release & Silk-Release
**No Manifest Changes**

### 2.6.0 CF-Networking-Release & Silk-Release
**No Manifest Changes**

### 2.5.0 CF-Networking-Release
**New Properties**
  - An optional paramater has been added to the `bosh-dns-adapter` job to configure custom internal domains. Defaults to `[apps.internal.]`
    - `internal_domains`

### 2.4.0 CF-Networking-Release
**No Manifest Changes**

### 2.3.0 CF-Networking-Release
**New Properties**
  - An optional parameter has been added to the `policy-server` and `policy-server-internal` jobs to configure the max number of
    open and idle connections to the policy-server database. Before making these configurable both properties were set to have no effective
    maximum. They now both default to 200 connections.
    - `max_open_connections`
    - `max_idle_connections`
  - An optional parameter has been added to the `policy-server-internal` job to configure the consul dns health check timeout.
    Defaults to 5 seconds.
    - `health_check_timeout_seconds`

**Changed Properties**
  - Namespaced the `connect_timeout_seconds` under `database` in the `policy-server` and `policy-server-internal` jobs.

### 2.3.0 Silk-Release
**New Properties**
  - An optional parameter has been added to the `silk-controller` job to configure the consul dns health check timeout. Defaults
    to 5 seconds.
    - `health_check_timeout_seconds`

**Changed Properties**
  - Namespaced the `connect_timeout_seconds` under `database` in the `silk-controller` job.

### 2.2.0 CF-Networking-Release
**New Properties**
  - A set of new optional properties have been added to the `policy-server` and `policy-server-internal` jobs to support the `pxc-release`.
    They requires using `pxc-release`. Both values *must* be set in the `policy-server` job to use TLS connections to the pxc database.
    `policy-server-internal` will get these values from `policy-server` via BOSH links.
    - `database.require_ssl` is a boolean flag for requiring TLS on the database connections.
    - `database.ca_cert` is the ca of the `pxc-release` database job.

### 2.2.0 Silk-Release
**New Properties**
  - A set of new optional properties have been added to the `silk-controller` job to support the `pxc-release`.
    They requires using `pxc-release`. Both values *must* be set to use TLS connections to the pxc database.
    - `database.require_ssl` is a boolean flag for requiring TLS on the database connections.
    - `database.ca_cert` is the ca of the `pxc-release` database job.

### 2.1.0 Silk-Release
**New Properties**
  - An optional parameter `host_tcp_services` has been added to the `silk-cni` job to specify TCP addresses
    running on the BOSH VM that should be accessible from containers. The address must not be in the 127.0.0.0/8
    range. The `silk-cni` plugin will install an iptables INPUT rule for each service.

### 2.0.0 CF-Networking-Release

**Silk No Longer Included in cf-networking-release**

The following jobs `cni (renamed: silk-cni)`, `iptables-logger`, `silk-controller`,
`vxlan-policy-agent`, `silk-daemon`, `netmon` have been moved to
[silk-release](code.cloudfoundry.org/silk-release). As a result, the properties for those jobs have been moved also:
- Job `cni` -> Renamed to `silk-cni`
  - `cf_networking.disable` -> `disable`
  - `cf_networking.mtu` -> `mtu`
  - `cf_networking.silk_daemon.listen_port` -> `silk_daemon.listen_port`
  - `cf_networking.iptables_logging` -> `iptables_logging`
  - `cf_networking.dns_servers` -> `dns_servers`
  - `cf_networking.rate` -> `rate`
  - `cf_networking.burst` -> `burst`
  - `cf_networking.iptables_denied_logs_per_sec` -> `iptables_denied_logs_per_sec`
  - `cf_networking.iptables_accepted_udp_logs_per_sec` -> `iptables_accepted_udp_logs_per_sec`
- Job `iptables-logger`
  - `cf_networking.iptables_logger.kernel_log_file` -> `kernel_log_file`
  - `cf_networking.iptables_logger.metron_port` -> `metron_port`
- Job `silk-controller`
  - `cf_networking.silk_controller.metron_address` -> `metron_address`
  - `cf_networking.silk_controller.poll_interval` -> `poll_interval`
  - `cf_networking.silk_controller.interface_name` -> `interface_name`
  - `cf_networking.silk_controller.log_level` -> `log_level`
  - `cf_networking.silk_controller.disable` -> `disable`
- Job `silk-controller`
  - `cf_networking.disable` -> `disable`
  - `cf_networking.network` -> `network`
  - `cf_networking.subnet_prefix_length` -> `subnet_prefix_length`
  - `cf_networking.subnet_lease_expiration_hours` -> `subnet_lease_expiration_hours`
  - `cf_networking.silk_controller.debug_port` -> `debug_port`
  - `cf_networking.silk_controller.health_check_port` -> `health_check_port`
  - `cf_networking.silk_controller.connect_timeout_seconds` -> `connect_timeout_seconds`
  - `cf_networking.silk_controller.listen_ip` -> `listen_ip`
  - `cf_networking.silk_controller.listen_port` -> `listen_port`
  - `cf_networking.silk_controller.ca_cert` -> `ca_cert`
  - `cf_networking.silk_controller.server_cert` -> `server_cert`
  - `cf_networking.silk_controller.server_key` -> `server_key`
  - `cf_networking.silk_controller.metron_port` -> `metron_port`
  - `cf_networking.silk_controller.database.type` -> `type`
  - `cf_networking.silk_controller.database.username` -> `username`
  - `cf_networking.silk_controller.database.password` -> `password`
  - `cf_networking.silk_controller.database.host` -> `host`
  - `cf_networking.silk_controller.database.port` -> `port`
  - `cf_networking.silk_controller.database.name` -> `name`
  - `cf_networking.silk_controller.max_open_connections` -> `max_open_connections`
  - `cf_networking.silk_controller.max_idle_connections` -> `max_idle_connections`
- Job `silk-daemon`
  - `cf_networking.disable` -> `disable`
  - `cf_networking.vtep_port` -> `vtep_port`
  - `cf_networking.rep_listen_addr_admin` -> `rep_listen_addr_admin`
  - `cf_networking.partition_tolerance_hours` -> `partition_tolerance_hours`
  - `cf_networking.lease_poll_interval_seconds` -> `lease_poll_interval_seconds`
  - `cf_networking.silk_daemon.vxlan_interface` -> `vxlan_interface`
  - `cf_networking.silk_daemon.ca_cert` -> `ca_cert`
  - `cf_networking.silk_daemon.client_cert` -> `client_cert`
  - `cf_networking.silk_daemon.client_key` -> `client_key`
  - `cf_networking.silk_daemon.listen_port` -> `listen_port`
  - `cf_networking.silk_daemon.debug_port` -> `debug_port`
  - `cf_networking.silk_daemon.metron_port` -> `metron_port`
  - `cf_networking.silk_controller.hostname` -> `silk_controller.hostname`
  - `cf_networking.silk_controller.listen_port` -> `silk_controller.listen_port`
- Job `vxlan_policy_agent`
  - `cf_networking.disable` -> `disable`
  - `cf_networking.iptables_logging` -> `iptables_logging`
  - `cf_networking.policy_server.hostname` -> `policy_server.hostname`
  - `cf_networking.policy_server.internal_listen_port` -> `policy_server.internal_listen_port`
  - `cf_networking.policy_poll_interval_seconds` -> `policy_poll_interval_seconds`
  - `cf_networking.vxlan_policy_agent.ca_cert` -> `ca_cert`
  - `cf_networking.vxlan_policy_agent.client_cert` -> `client_cert`
  - `cf_networking.vxlan_policy_agent.client_key` -> `client_key`
  - `cf_networking.vxlan_policy_agent.metron_port` -> `metron_port`
  - `cf_networking.vxlan_policy_agent.debug_server_port` -> `debug_server_port`
  - `cf_networking.vxlan_policy_agent.log_level` -> `log_level`
  - `cf_networking.iptables_accepted_udp_logs_per_sec` -> `iptables_accepted_udp_logs_per_sec`
- Job `netmon`
  - `cf_networking.disable` -> `disable`
  - `cf_networking.netmon.metron_address` -> `metron_address`
  - `cf_networking.netmon.poll_interval` -> `poll_interval`
  - `cf_networking.netmon.interface_name` -> `interface_name`
  - `cf_networking.netmon.log_level` -> `log_level`

**Changed Properties**
- ALL properties from ALL jobs in this release have had their namespaces removed.
  The `cf_networking.<job_name>` prefixes are no longer necessary given bosh supports job level properties.
- Removed `policy-server-internal.tag_length` property from `policy-server-internal` job, this property is retrieved via bosh links from
  `tag_length is used by link to policy-server.tag_length`.
- To support third party integrators we have changed the cni job in `silk-release` to `silk-cni`
  - To continue using `silk` the `cni_plugin_dir` and `cni_config_dir` on the `garden-cni`
    job must be explicitly set in the manifest as follows:
    - `cni_plugin_dir: /var/vcap/packages/silk-cni/bin`
    - `cni_config_dir: /var/vcap/jobs/silk-cni/config/cni`
  - Third party integrators should continue using the default values for these properties
- `connect_timeout_seconds` on the `policy-server` and `policy-server-internal` jobs now affects
  how long it takes for a db connection to be established when the policy server starts. Before
  this was hardcoded to a timeout of 5 seconds.

### 2.0.0 Silk-Release
**New Properties**
  - An optional parameter `no_masquerade_cidr_range` has been added to the `silk-cni` job to specify which destination
    CIDR to exempt MASQUERADEing traffic from containers.
    If this is left unset and the bosh link `cf_network` is available with the property `network` set, it will use that value.
    Otherwise, an empty default value will be applied. If empty it will not exclude any ranges.
  - Add `disable` property to all jobs. When it is set to true the job will not start.
  - To support third party integrators we have changed the `cni` job in `silk-release` to `silk-cni`
    - To continue using `silk` the `cni_plugin_dir` and `cni_config_dir` on the `garden-cni`
      job must be explicitly set in the manifest as follows:
      - `cni_plugin_dir: /var/vcap/packages/silk-cni/bin`
      - `cni_config_dir: /var/vcap/jobs/silk-cni/config/cni`

### 1.13.0 CF-Networking-Release
**New Properties**
  - An optional parameter has been added to the `garden-cni` job to
    specify search domains. These domains will be configured in containers' /etc/resolv.conf.
    - `cf_networking.search_domains`
  - An optional parameter has been added to the `silk-daemon` job to configure which network
    container traffic should be sent over based on network interface name. This property is
    not recommended for use and is temporary. If empty, the default network is used.
    - `cf_networking.silk_daemon.temporary_vxlan_interface`
  - An optional parameter has been added to the `silk-daemon` job to configure which network
    container traffic should be sent over based on bosh network name. If empty, the default
    gateway network is used.
    - `cf_networking.silk_daemon.vxlan_network`
  - An optional parameter has been added to list domains from which Cross-Origin
    requests will be accepted.
    - `cf_networking.policy_server.allowed_cors_domains`

### 0.2.0 Silk-Release (Tested with CF-Networking-Release v1.12.0)
**New Properties**
  - An optional parameter has been added to the `silk-daemon` job to specify which bosh network should be used by the
    vxlan adapter.
    The property value is expected to be the name of a bosh network that is attached to the instance group where the
    `silk-daemon` is running.
    This is useful when running multi-homed vms. If this property is not specified, the bosh network that is the default
    gateway will be chosen.
    - `vxlan_network`

### 1.11.0 CF-Networking-Release
**Changed Properties**
  - `cf_networking.silk_controller.connect_timeout_seconds` now defaults to 120.
  - `cf_networking.policy_server.connect_timeout_seconds` now defaults to 120.
  - `cf_networking.policy_server_internal.connect_timeout_seconds` now defaults to 120.

**New Properties**
  - An optional parameter has been added to determine what interface the silk-vtep should
    attach to based on BOSH network name. If not set, we use the network that the BOSH spec
    defaults to. You cannot set this property and the `temporary_vxlan_interface` property together.
    - `cf_networking.silk_daemon.vxlan_network`
  - An optional parameter has been added to determine what interface the silk-vtep should
    attach to based on interface name. We do not recommend using this parameter and it is
    temporary. You cannot set this property and the `vxlan_network` property together.
    - `cf_networking.silk_daemon.temporary_vxlan_interface`

### 0.1.0 Silk-Release (Tested with CF-Networking-Release v1.11.0)
  - This release was extracted from [cf-networking-release](github.com/cloudfoundry/cf-networking-release).
    Refer to that release for prior changes.

### 1.7.0 CF-Networking-Release
**New Properties**
  - An optional parameter has been added to turn on bosh backup and restore.
    By default, this property is set to false and backup and restore is turned off.
    - `release_level_backup`
  - An optional parameter has been added to configure the max number of
    open and idle connections to the silk-controller database.
    - `cf_networking.silk_controller.max_open_connections`
    - `cf_networking.silk_controller.max_idle_connections`

### 1.6.0 CF-Networking-Release

**Changed Properties**

  - The value for `cf_networking.garden_external_networker.cni_plugin_dir` now defaults to `/var/vcap/packages/cni/bin`
  - The value for `cf_networking.garden_external_networker.cni_config_dir` now defaults to `/var/vcap/jobs/cni/config/cni`


### 1.5.0 CF-Networking-Release
**Links Enabled**
The `policy-server` now provides database connection info via a link which the new `policy-server-internal` job consumes:
  - `cf_networking.policy_server.database.type`
  - `cf_networking.policy_server.database.username`
  - `cf_networking.policy_server.database.password`
  - `cf_networking.policy_server.database.port`
  - `cf_networking.policy_server.database.name`
  - `cf_networking.policy_server.database.host`

**New Properties**
  - REQUIRED: A new job `policy-server-internal` has been added. This job requires the following properties:
    - `cf_networking.policy_server_internal.ca_cert`
    - `cf_networking.policy_server_internal.server_cert`
    - `cf_networking.policy_server_internal.server_key`
    There are additional optional paramaters that can be set and are viewable in [the spec file](../jobs/policy-server-internal/spec)
  - An optional parameter has been added to configure the path to the iptables kernel log for
    the iptables_logger.
    - `cf_networking.iptables_logger.kernel_log_file`

**Removed Properties**
  - The `policy-server` job has removed the following properties:
    - `cf_networking.policy_server.internal_listen_port`
    - `cf_networking.policy_server.ca_cert`
    - `cf_networking.policy_server.server_cert`
    - `cf_networking.policy_server.server_key`

**Changed Properties**
  - The `consul.agent.services.policy-server` property for the `consul_agent` job on the `api` instance group
    should be renamed to `consul.agent.services.policy-server-internal`.

### 1.4.0 CF-Networking-Release
**Links Enabled**
The `silk-controller` job now provides two properties via links which the `silk-daemon` job consumes:

- `cf_networking.network`
- `cf_networking.subnet_prefix_length`
** This means you are able to remove the properties (listed above) from the `silk-daemon` job. **

If your deployment contains more than a single instance group that has the `silk-controller` job,
then you will need to explicitly name the `cf_network` link. For more information,
[see the documentation](https://bosh.io/docs/links.html#deployment).

**New Properties**
  - An optional parameter has been added to configure the port of the metron agent for
    the iptables_logger. This port will be used to forward metrics. Previously, no such
    port existed.
    - `cf_networking.iptables_logger.metron_port`


### 1.3.0 CF-Networking-Release

**New Properties**
  - An optional parameter has been added to configure the rate of logs by
    iptables for accepted UDP packets. Before, logging was done per UDP
    connection. Now, the rate defaults to 100 packets per second.
    - `cf_networking.iptables_accepted_udp_logs_per_sec` is the maximum number of
      accepted udp packets logged by iptables per second, it should be
      configured on the `silk-cni` job for ASGs or on the `vxlan-policy-agent`
      job for C2C.

### 1.2.0 CF-Networking-Release

**New Properties**

  - Optional parameters have been added to the `silk-cni` job to limit the
    bandwidth in and out of containers.
    - `cf_networking.rate` is the rate in Kbps at which traffic can leave and
      enter a container.
    - `cf_networking.burst` is the burst in Kb at which traffic can leave and
      enter a container.
    - Both of these parameters must be set in order to limit bandwidth. If
      neither one is set, then bandwidth is not limited.
    - The burst must high enough to support the given rate. If burst is not
      high enough, then creating containers will fail.
  - An optional parameter has been added to configure the rate of logs by
    iptables for denied packets. Before, this rate was hardcoded to 2 packets
    per minute. Now, the rate defaults to 1 packet per second.
    - `cf_networking.iptables_denied_logs_per_sec` is the maximum number of
      denied packets logged by iptables per second, it should be configured on
      the `silk-cni` job.

### 1.1.0 CF-Networking-Release

**New Properties**

  - An optional parameter has been added to allow all space developers to create policies (default `false`).
    If this property is not set, a space developer must have `network.write` to create policies.
    - `cf_networking.enable_space_developer_self_service`
  - An optional parameter has benn added to configure the maximum number of policies a space
    developer can write for a given source app. Defaults to 50 if it is not set. Does not apply to
    users with `network.admin`:
    - `cf_networking.max_policies_per_app_source`
    
### 1.0.0 CF-Networking-Release

**New Properties**

  - The following optional parameters have been added to override the timeout values for
    database connections and DNS health checks for the silk controller and policy server:
    - `cf_networking.silk_controller.connect_timeout_seconds`
    - `cf_networking.policy_server.connect_timeout_seconds`

  - This optional property has been added to override the metron port on the silk controller:
    - `cf_networking.silk_controller.metron_port`

  - This optional property has been added to override the health check port on the silk controller:
    - `cf_networking.silk_controller.health_check_port`

**Removed Properties**

  - The following properties have been removed from the silk-controller job,
    **but still must be set on the silk-daemon job**.
    -  `cf_networking.silk_daemon.ca_cert`
    -  `cf_networking.silk_daemon.client_cert`
    -  `cf_networking.silk_daemon.client_key`

### 0.25.0 CF-Networking-Release

**New Properties**

  - The optional parameter `cf_networking.lease_poll_interval_seconds` has been added to allow
    operators to override the default polling interval between silk-daemon and silk-controller.

**Changed Properties**

  - The value for `cf_networking.garden_external_networker.cni_config_dir` now defaults to `/var/vcap/jobs/silk-cni/config/cni`
    We recommend that you remove any overrides for this property, unless you are intending to use a 3rd party CNI plugin.

**Other Changes**

Since silk is now deployed by default, there is no more `silk.yml` ops file.  Deploying with flannel is no longer supported.

### 0.24.0 CF-Networking-Release

**New Properties**

The host port for receiving VXLAN packets is now configurable as `cf_networking.vtep_port` for flannel and silk.
Overriding this value is optional.

### 0.22.0 CF-Networking-Release

This release introduces a new container networking fabric called "silk" and
**contains significant changes to job and property names**.
Silk will become the default networking fabric in our next release.

Silk is a replacement for flannel, which uses a central controller node backed a SQL database.
Etcd is no longer required by CF Networking Release when running Silk.

When deploying with Silk, the following new jobs will be added:

- On the Diego cells: `silk-cni` and `silk-daemon`
- On the Diego BBS VM: `silk-controller`

These jobs require new certificates and a new (logical) database (separate from the policy server database).

We recommend you review the [spec files for these new jobs](../jobs) and the diff below.

The `cni-flannel` job will no longer be running on Diego cells.

To deploy Silk with [CF Deployment](https://github.com/cloudfoundry/cf-deployment), use the
[`silk.yml` opsfile](../manifest-generation/opsfiles/silk.yml) as documented for [GCP, AWS and BOSH-lite](https://github.com/cloudfoundry/cf-deployment).

Instructions for deploying with CF Release have also been updated in the above docs.  Your stub file
should have a diff that resembles:
```diff
   driver_templates:
   - name: garden-cni
     release: cf-networking
-  - name: cni-flannel
+  - name: silk-cni
+    release: cf-networking
+  - name: silk-daemon
     release: cf-networking
   - name: netmon
     release: cf-networking
   - name: vxlan-policy-agent
     release: cf-networking
+  bbs_templates:
+  - name: silk-controller
+    release: cf-networking
+  bbs_consul_properties:
+    agent:
+      services:
+        silk-controller: {}
   properties:
     cf_networking:
-      garden_external_networker:
-        cni_config_dir: /var/vcap/jobs/cni-flannel/config/cni
+      cni_config_dir: /var/vcap/jobs/silk-cni/config/cni
+      silk_controller:
+        database:
+          username: networkconnectivityadmin
+          password: admin
+          host: 10.244.0.30
+          port: 5524
+          name: networkconnectivitydb
+          type: postgres
+        ca_cert: |
+          -----BEGIN CERTIFICATE-----
+          REPLACE
+          -----END CERTIFICATE-----
+        server_cert: |
+          -----BEGIN CERTIFICATE-----
+          REPLACE
+          -----END CERTIFICATE-----
+        server_key: |
+          -----BEGIN RSA PRIVATE KEY-----
+          REPLACE
+          -----END RSA PRIVATE KEY-----
+      silk_daemon:
+        ca_cert: |
+          -----BEGIN CERTIFICATE-----
+          REPLACE
+          -----END CERTIFICATE-----
+        client_cert: |
+          -----BEGIN CERTIFICATE-----
+          REPLACE
+          -----END CERTIFICATE-----
+        client_key: |
+          -----BEGIN RSA PRIVATE KEY-----
+          REPLACE
+          -----END RSA PRIVATE KEY-----
-      plugin:
-        etcd_endpoints:
-          - (( config_from_cf.etcd.advertise_urls_dns_suffix ))
-        etcd_client_cert: (( config_from_cf.etcd.client_cert ))
-        etcd_client_key: (( config_from_cf.etcd.client_key ))
-        etcd_ca_cert: (( config_from_cf.etcd.ca_cert ))
```

### 0.21.0 CF-Networking-Release

**Changed Properties**

  - The value for `cf_networking.garden_external_networker.cni_plugin_dir` now defaults to `/var/vcap/packages/silk-cni/bin`
    We recommend that you remove any overrides for this property, unless you are intending to use a 3rd party CNI plugin.

### 0.20.0 CF-Networking-Release

**Changed Properties**

  - The value for `cf_networking.garden_external_networker.cni_plugin_dir` now defaults to `/var/vcap/packages/silk/bin`
    We recommend that you remove any overrides for this property, unless you are intending to use a 3rd party CNI plugin.

### 0.19.0 CF-Networking-Release

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


### 0.18.0 CF-Networking-Release

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

### 0.17.0 CF-Networking-Release
Policy server requires a CA cert for UAA, **manifest must be generated with `diego-release` v1.7.0+**

The following needs to be added to your `cf-networking` stub **even if you are skipping ssl validation of UAA**:

```diff
cf_networking_overrides:
  properties:
    cf_networking:
      policy_server:
+      uaa_ca: (( config_from_cf.uaa.ca_cert ))
```

### 0.15.0 CF-Networking-Release

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
Refer to the [cf-deployment docs](https://github.com/cloudfoundry/cf-deployment)
for details on how to generate and configure certs and keys.
Note, you will likely need to make similar changes to other etcd clients, e.g. loggregator.

0. In the stub file, `netman_overrides` renamed to `cf_networking_overrides`

### 0.12.0 CF-Networking-Release

In the CF properties stub:

```diff
    -   scope: cloud_controller.read,cloud_controller.write,openid,password.write,cloud_controller.admin,scim.read,scim.write,doppler.firehose,uaa.user,routing.router_groups.read,network.admin
    +   scope: cloud_controller.read,cloud_controller.write,openid,password.write,cloud_controller.admin,scim.read,scim.write,doppler.firehose,uaa.user,routing.router_groups.read,network.admin,network.write
```

###  0.10.0 CF-Networking-Release
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

###  0.7.0 CF-Networking-Release

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
