# Known Issues

### IPTables Logging

  We are investigating known issues with iptables logging. It is currently non-functional.
  This work can be followed on [tracker](https://www.pivotaltracker.com/story/show/156589146).
  Changes to the syslog-release are incompatible with the way that logs are generated.

### Compatibility with VMware NSX for vSphere 6.2.3+

  When using VMware NSX for vSphere 6.2.3+, the default VXLAN port of 4789 used by cf-networking will not work.
  To fix this issue, override the default `cf_networking.vtep_port` with another value.

### MySQL versions below 5.7

  When the policy server is backed by MySQL versions < 5.7, a user may see this error when trying to create a policy:

  ```
  FAILED
  adding policies: failed to make request to policy server
  ```

  An operator inspecting the logs of the `policy-server` BOSH job may see this error:

  > creating destination: Error 1064: You have an error in your SQL syntax;
  check the manual that corresponds to your MySQL server version for the right
  syntax to use near 'WHERE\n\t\tNOT EXISTS (\n\t\t\tSELECT *\n\t\t\tFROM destinations\n\t\t\tWHERE group_id = ? AND '

  This issue can be resolved by upgrading your MySQL server to version 5.7+

### Container network access may require a one-time app restage
  Apps which were last pushed or restaged on older versions of CloudController
  may need to be restaged on a newer version of CloudController in order to
  connect to other apps via the container network.

  Apps which have been pushed or restaged on [capi-release v1.0.0](https://github.com/cloudfoundry/capi-release/releases/tag/v1.0.0)
  or higher, or [cf-release v240](https://github.com/cloudfoundry/cf-release/releases/tag/v240) or higher
  should be ok.

  One symptom of this issue is frequent log messages from the `vxlan-policy-agent` job on
  the Diego Cell VMs which include the message
  ```
  Container metadata is missing key policy_group_id. Check version of CloudController.
  ```

  To resolve this, simply `cf restage MYAPP`.


### Missing Feature Parity For Application Security Groups
  Logging for UDP and ICMP ASGs is currently not supported, but [this feature is on our roadmap](https://www.pivotaltracker.com/story/show/142629505).

###  Behavior Changes From Existing Application Security Groups
  Prior implementations of ASGs allowed opening security groups to other containers
  via the NATed port on the diego cell.  With CF Networking, this is no longer supported.
  Direct addressing of other containers is only possible through the overlay network.

### Blue/Green deploys of apps must reconfigure policies
  Following the instructions
  [here](https://docs.cloudfoundry.org/devguide/deploy-apps/blue-green.html),
  when the green app is deployed it will have a different app guid than blue,
  meaning any container to container policies that blue has configured will need
  to be configured for green as well.
