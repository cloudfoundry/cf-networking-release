# Known Issues

### Compatibility with VMware NSX for vSphere 6.2.3+

  When using VMware NSX for vSphere 6.2.3+, the default VXLAN port of 4789 used
  by cf-networking will not work.  To fix this issue, override the default
  `cf_networking.vtep_port` with another value.

### Container network access may require a one-time app restage
  Apps which were last pushed or restaged on older versions of CloudController
  may need to be restaged on a newer version of CloudController in order to
  connect to other apps via the container network.

  Apps which have been pushed or restaged on [capi-release
  v1.0.0](https://github.com/cloudfoundry/capi-release/releases/tag/v1.0.0) or
  higher, or [cf-release
  v240](https://github.com/cloudfoundry/cf-release/releases/tag/v240) or higher
  should be ok.

  One symptom of this issue is frequent log messages from the
  `vxlan-policy-agent` job on the Diego Cell VMs which include the message

  ```
  Container metadata is missing key policy_group_id. Check version of CloudController.
  ```

  To resolve this, simply `cf restage MYAPP`.

###  Behavior Changes From Existing Application Security Groups

  Prior implementations of ASGs allowed opening security groups to other
  containers via the NATed port on the diego cell.  With CF Networking, this is
  no longer supported.  Direct addressing of other containers is only possible
  through the overlay network.

### Blue/Green deploys of apps must reconfigure policies

  Following the instructions
  [here](https://docs.cloudfoundry.org/devguide/deploy-apps/blue-green.html),
  when the green app is deployed it will have a different app guid than blue,
  meaning any container to container policies that blue has configured will need
  to be configured for green as well.

### Silk-Release IPTables logging not effective in BOSH Lite

  When using BOSH Lite, iptables logging between deployed applications on the
  same diego-cell does not log to `/var/log/kern.log`. Due to this, the
  iptables-logger job is unable to read and generate the logging statements in
  `/var/vcap/sys/log/iptables-logger/iptables.log`.

### When using bosh-lite, not finding iptable logging inside kern.log
The linux kernel prevents iptable log targets from working inside a container.
See [commit introducing the
change](https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/commit/?id=69b34fb996b2eee3970548cf6eb516d3ecb5eeed)

