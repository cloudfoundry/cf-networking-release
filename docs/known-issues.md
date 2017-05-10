# Known Issues

### Compatibility with VMware NSX for vSphere 6.x

  When using VMware NSX for vSphere 6.x, the default VXLAN port of 8472 used by cf-networking is not allowed.
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
  Current support for application security groups in cf-networking is incomplete:
  - The only supported protocols are `tcp` and `udp`, this means `icmp` protocol,
    code and type are not supported

###  Behavior Changes From Existing Application Security Groups
  Current implementations of ASGs allow opening security groups to other containers
  via the NATed port on the diego cell. With container networking we only support
  direct addressing of other containers through the overlay network and app-to-app
  policy system. Direct addressing of other containers (without going through the gorouter)
  on the underlay is not supported and may result in undefined behavior.


### Blue/Green deploys of apps must reconfigure policies
  Following the instructions
  [here](https://docs.cloudfoundry.org/devguide/deploy-apps/blue-green.html),
  when the green app is deployed it will have a different app guid than blue,
  meaning any container to container policies that blue has configured will need
  to be configured for green as well.
