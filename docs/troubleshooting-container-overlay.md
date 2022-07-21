# Troubleshooting Container Overlay Networking

NOTE: If you are having problems, first consult our [known issues
doc](known-issues.md).

This guide relates to troubleshooting problems between host + container networking,
primarily when using the Silk CNI. Some concepts can be used for other CNIs directly.
Others might require slight adaptations

### Checking Logs

* Discovering All CF Networking Logs:

  All cf-networking components log lines are prefixed with `cfnetworking` (no
  hyphen) followed by the component name. To find all CF Networking logs, run:

  ```bash
  grep -r cfnetworking /var/vcap/sys/log/*
  ```

  The log lines for the following components will be returned:
  * `silk-daemon` (from [silk-release](https://github.com/cloudfoundry/silk-release))
  * `silk-controller` (from [silk-release](https://github.com/cloudfoundry/silk-release))
  * `vxlan-policy-agent` (from [silk-release](https://github.com/cloudfoundry/silk-release))
  * `policy-server`
  * `policy-server-internal`
  * `netmon` (from [silk-release](https://github.com/cloudfoundry/silk-release))
  * `silk-cni` (from [silk-release](https://github.com/cloudfoundry/silk-release))
  * `garden` (from [garden-runc-release](https://github.com/cloudfoundry/garden))

  The log lines for the following components will be not returned:
  * `iptables`(from [silk-release](https://github.com/cloudfoundry/silk-release)): We have limited
    room to add an identifier, and these logs are high-volume.
  * `garden-external-networker`: Garden will only print errors if creating a
    container fails.
  * `cni-wrapper-plugin`(from [silk-release](https://github.com/cloudfoundry/silk-release)): Only
    stdout and stderr are printed in garden logs. However the call to the underlying silk-cni *will*
    log to `/var/vcap/sys/log/silk-cni/silk-cni.stdout.log`. It defaults to only logging error messages,
    so debugging may be needed.

* Container Create is Failing:

  If container create is failing check the garden logs, located on the cell VMs
  at `/var/vcap/sys/log/garden/garden.stdout.log`.  Garden logs stdout and
  stderr from calls to the CNI plugin, you can find any errors related to the
  CNI ADD/DEL there.

  Search for `external-network` or `CNI`, and look for messages related to setting up the container.
  There will also likely be results for failures to tear down the container - ignore those. Garden
  will attempt to destroy any failed resources it might have created, so if the create failed, this
  destroy will also likely fail. Focus on the initial create.

  Unsuccessful create will say things like `exit status 1` in the `stderr` field
  of the log message.

* Problems With ASG Creation:

  Problems applying egress iptables rules for silk-based containers will show up in the vxlan-policy-agent logs
  at `/var/vcap/sys/log/vxlan-policy-agent/vxlan-policy-agent.stdout.log`. 

  If IPTables rule application is successful, but the rules are incorrect, also check the `policy-server-internal` logs on
  the VM(s) hosting that server at `/var/vcap/sys/log/policy-server/policy-server-internal.stdout.log`, as well as the
  `policy-server-asg-syncer` logs in `/var/vcap/sys/log/policy-server-asg-syncer/policy-server-asg-syncer.stdout.log`.
  it 

### Enabling Debug Logging

Most components log at the `info` level by default. In many cases, the log level can be
adjusted at runtime by making a request to the debug server running on the VM.
To enable debug logging ssh to the VM and make this request to the debug server:

```bash
curl -X POST -d 'DEBUG' localhost:31821/log-level
```

To switch back to info logging make this request:

```bash
curl -X POST -d 'INFO' localhost:31821/log-level
```

For the policy server, the debug server listens on port 31821 by default, it can
be overridden by the manifest property `cf_networking.policy_server.debug_port`.

To enable debug logging for the `silk-cni`, create the `/var/vcap/jobs/silk-cni/config/enable_debug`
file on the `diego-cell` VM. Subsequent `silk-cni` calls will then log with debugging. 
**NOTE:** This will generate a **lot** of logging during container startup/teardown. Do this with caution.

## Diagnosing Why a Container Cannot Communicate

### Common Scenarios

#### `cf ssh` and app ingress work but some or all egress from the container does not

This is likely an issue with the ASGs assigned to the container. It could indicate one of the following problems
with Dynamic ASGs:

* Not getting the correct IPTables rules applied
  * Investigate the logs for `policy-server-asg-syncer`, and `vxlan-policy-agent` for issues encountered syncing
    rule data, or applying IPTables rules.
* Not having the correct ASGs associated with the app's space
  * Review the global ASGs for the container type (running or staging), as well as what ASGs are applied to
    the container's space (again either for running or staging). Create or apply the necessary ASGs for the
    failing traffic.
* ASG rules not being what is necessary for the traffic to succeed.
  * Review IPTables rules on the container, and compare them with the egress traffic that is failing. Add
    to or adjust the ruleset in the ASG definitions, restart the app (unless dynamic ASGs are enabled), and
    try again.

#### `cf ssh` does not work, but app ingress works.

This is likely an issue upstream from the cell, related to `cf ssh`. There is a small chance that this is related to
container networking. See [validating cf ssh](#validating-cf-ssh) for troubleshooting steps.

#### App ingress, egress, and `cf ssh` does not work on some or all containers

This case is extremely rare, but might be seen when running on an untested/beta stemcell. There is definitely a problem
with the networking for the application container(s). Validate [host-side networking](#validating-host-side-networking-when-using-silk-release)
and [container-side networking](#validating-container-side-networking-when-using-silk-release) to determine what the issue
is, and open an issue against the container's CNI (e.g. [silk-release](https:github.com/cloudfoundry/silk-release)).

### Troubleshooting Steps

#### Validating `cf ssh`

1. Run `cf space-ssh-allowed` and `cf ssh-enabled <app>` to ensure that app ssh is enabled. If not, use the `cf` CLI to enable
   SSH for the app.
1. `bosh ssh` into the diego-cell hosting the container.
2. Use `cfdot actual-lrps` and `jq` to find the `instance_id` of the container you are trying to SSH into.
2. Use `iptables -S -t nat | grep <first-octet-of-container-instance-id` to identify the port-forwarding rules that translate 
   the `ssh traffic` between the host's`--dport` (>61000) and `--to-destination 10.255.x.x:61002` or `2222`. Note the value
   of the `--dport` flag.
3. Run a tcpdump to determine if the cell receives any trafficd on the port determined above. If not, it indicates the issue
   is likely upstream of cf-networking. If there is traffic received, there is likely an issue with the container.
4. For issues upstream, review diego's `ssh_proxy` job logs, and the configuration of the loadbalancer used for app SSH.
5. If step 3 determined that SSH traffic was reaching the host, but not going through, validate [host-side networking](#validating-host-side-networking-using-silk-release) and
   [container-side networking](#validating-container-side-networking-using-silk-release).

#### Validating Host-Side Networking when using [silk-release](https://github.com/cloudfoundry/silk-release)

1. `bosh ssh` into the diego-cell hosting the container.
2. Use `cfdot actual-lrps` and `jq` to find the `instance_guid` and `instance_address` of the container you are trying to SSH into.
3. Use `ip addr show | grep -B3 <container ip>` to find the interface name, MAC address, and namespace id of the host-side interrface. Validate that the MAC
   address begins with `aa:aa:<hex-encoded-container-ip>`. Validate that the interface name matches `s-<zero-padded-container-ip>`.
   For example:
```
$ ip addr show | grep -B2 10.255.211.40
64376: s-010255211040@if64375: <BROADCAST,MULTICAST,NOARP,UP,LOWER_UP> mtu 1410 qdisc noqueue state UP group default
    link/ether aa:aa:0a:ff:d3:28 brd ff:ff:ff:ff:ff:ff link-netnsid 0
    inet 169.254.0.1 peer 10.255.211.40/32 scope link s-010255211040
```
   The namespace id is `0`, obtained from `link-netnsid 0`. `s-010255211040` is the interface name, and `aa:aa:0a:ff:d3:28` is the MAC address (`0a` is hex for `10`, `ff` is hex for `255`,
   `d3` is hex for 211, and `28` is hex for `40`). If the interface name does not match the IP, or MAC does not match `aa:aa:<hex-endoded-ip-addr>`
   something is wrong with the way the overlay bridge was set up in the `silk-cni` binary. Review `silk-cni` logs for any errors,
   or enable debugging on `silk-cni` for more information.
4. Use `arp -a | grep <container ip>` to validate that the ARP table has an entry pointing the container IP through the interface
   name obtained above, using a MAC addr of `ee:ee:<hex-encoded-container-ip>`. If this entry is incorrect or missing, there
   was an issue in `silk-cni` setting up the overlay bridge. Review `silk-cni` logs for any erors, or enable debugging on `silk-cni`
   for more information.
5. The IP address of the `s-<zero-padded-container-ip>` interface should be 169.254.0.1, and should **ALWAYS** match the default
   gateway denfined iside the container ([see validating container-side-networking](#validating-container-side-networking-when-using-silk-release)).
6. If everything else looks good, validate that the namespace ID for the container processes match the namespace ID for the host's
   `s-<zero-padded-container-ip>` interface:
   1. Run `ps -awxfu | less` to get a full host process-tree. Search the output for the container's `instance_guid` to find the
      parent `gdn` process. Scan down the `gdn` process's tree to find child processes for `diego-sshd`, `envoy`, and the app process.
      Not the process IDs of these three processes (second column of the output).
    2. Validate that all three processes share the same networking namespace inode reference by running `ls -l /proc/<pid>/ns/net`.
       It should show up as a link to `net:[<namespace inode>]`.
    3. Confirm that the namespace inode matches the namespace id obtained from the `s-<zero-padded-container-ip>` interface above:
      `lsns -l -t net | egrep 'NETNSID|<namespace inode>'`. The NETNSID column should reflect the namespace ID of the interface.

#### Validating Container-Side Networking when using [silk-release](https://github.com/cloudfoundry/silk-release)

1. `bosh-ssh` into the diego-cell hosting the container.
2. Run `ps -awxfu | less` to get a full host process-tree. Search the output for the container's `instance_guid` to find the
   parent `gdn` process. Scan down the `gdn` process's tree to find child processes for `diego-sshd`, `envoy`, and the app process.
   Not the process IDs of these three processes (second column of the output).
3. Validate that all three processes share the same networking namespace inode reference by running `ls -l /proc/<pid>/ns/net`.
   It should show up as a link to `net:[<namespace inode>]`.
4. Enter a bash shell as root in the container namespaces with `nsenter -t <app-pid> -a bash`.
5. Use `ip addr show` to validate that there is an interface named `c-<zero-padded-container-ip>`, with a MAC address of `ee:ee:<hex-encoded-ip-addr>`.
6. Use `arp -a` to validate an entry exists for 169.254.0.1 (or the `s-<zero-padded-container-ip>` IP addr found when [validating host-side
   networking](#validating-host-side-networking-when-using-silk-release)), and that the entry points to the same `aa:aa:<hex-encoded-ip-addr>` MAC address of that interface.
7. Use `netstat -rn` to ensure the default gateway of the container points to the IP addr listed in the `arp -a` output.
