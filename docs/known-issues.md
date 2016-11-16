# Known Issues

- ### MySQL versions below 5.7

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


- ### Missing Feature Parity For Application Security Groups
  Current support for application security groups in netman is incomplete:
  - The only supported protocols are `tcp` and `udp`, this means `icmp` protocol,
    code and type are not supported
  - Only single ports are supported, not ranges
  - We currently do not support logging in ASGs

- ###  Behavior Changes From Existing Application Security Groups
  Current implementations of ASGs allow opening security groups to other containers
  via the NATed port on the diego cell. With container networking we only support
  direct addressing of other containers through the overlay network and app-to-app
  policy system. Direct addressing of other containers (without going through the gorouter)
  on the underlay is not supported and may result in undefined behavior.

- ### CIDR blocks other than /16
  It is possible to configure the CIDR block for containers to be something
  other than the default of /16. This hasn't been tested.
  We don't know what happens. Good luck.

- ### Blue/Green deploys of apps must reconfigure policies
  Following the instructions
  [here](https://docs.cloudfoundry.org/devguide/deploy-apps/blue-green.html),
  when the green app is deployed it will have a different app guid than blue,
  meaning any container to container policies that blue has configured will need
  to be configured for green as well.


- ### Stale policies not cleaned up
  If you push an app and configure a policy for that app, when you delete the app
  without deleting the policy, then the policy will stay in the policies database and be seen when you run (as cf admin):
  ```bash
  cf curl /networking/v0/external/policies
  ```
  Under normal circumstances, these stale policies should not cause any major issues.
  If you want to delete every policy from the server run (as cf admin):
  ```bash
  cf curl /networking/v0/external/policies -X DELETE -d `cf curl /networking/v0/external/policies`
  ```

- ### Upgrading/Downgrading between garden-runc and garden-runc + netman requires a recreate
garden-runc and netman may leave around iptables rules or networking devices when switching networking stacks.
The safest way to upgrade from one to the other is to run:
  ```bash
  bosh deploy --recreate
  ```

- ### Flannel watchdog fails on bosh-lite
  Flannel on bosh-lite often gets into a state where the overlay network is not functioning.
  A process called `flannel-watchdog` runs on the cells and checks for this error and will cause BOSH to consider the VM unhealthy.
  If you run `bosh vms` and see output similar to this:
  ```
  Deployment 'cf-warden-diego'

  Director task 939

  Task 939 done

  +-----------------------------------------------------------+---------+-----+------------------+--------------+
  | VM                                                        | State   | AZ  | VM Type          | IPs          |
  +-----------------------------------------------------------+---------+-----+------------------+--------------+
  | access_z1/0 (6fa80b0e-eda8-468f-b7c6-e047445627be)        | running | n/a | access_z1        | 10.244.16.22 |
  | brain_z1/0 (6ad95643-b814-4bb4-9f55-d06cce9def8c)         | running | n/a | brain_z1         | 10.244.16.6  |
  | cc_bridge_z1/0 (791654b4-fbaa-4c51-8115-8ad1e8078846)     | running | n/a | cc_bridge_z1     | 10.244.16.14 |
  | cell_z1/0 (4d2a0aac-f136-4938-b7d2-e9b435e259b4)          | failing | n/a | cell_z1          | 10.244.16.10 |
  | database_z1/0 (cfa8424e-302d-44db-8fc6-11b5aecb5d70)      | running | n/a | database_z1      | 10.244.16.2  |
  | policy-server/0 (446c71f7-284c-4ec4-9a34-28c8b1e33edd)    | running | n/a | database_z1      | 10.244.16.26 |
  | route_emitter_z1/0 (77e16e90-21da-4a50-9597-61b31ad0c9cf) | running | n/a | route_emitter_z1 | 10.244.16.18 |
  +-----------------------------------------------------------+---------+-----+------------------+--------------+

  VMs total: 7
  ```

  And running `monit summary` on the cell shows:
  ```
  Process 'consul_agent'              running
  Process 'rep'                       running
  Process 'garden'                    running
  Process 'metron_agent'              running
  Process 'flanneld'                  running
  Process 'flannel-watchdog'          Does not exist
  Process 'netmon'                    running
  Process 'vxlan-policy-agent'        running
  System 'system_localhost'           running
  ```

  Then flannel is in an unrecoverable state and the cell job needs to be recreated:
  ```
  bosh recreate cell_z1
  ```
