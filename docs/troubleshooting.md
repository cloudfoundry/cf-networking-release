# Troubleshooting

NOTE: If you are having problems, first consult our [known issues doc](known-issues.md).

### Checking Logs

* Discovering All CF Networking Logs:

  All cf-networking components log lines are prefixed with `cfnetworking` (no hyphen)
  followed by the component name. To find all CF Networking logs, run:
  `grep -r cfnetworking /var/vcap/sys/log/*`

  The log lines for the following components will be returned:
  * `silk-daemon` (from [silk-release](code.cloudfoundry.org/silk))
  * `silk-controller` (from [silk-release](code.cloudfoundry.org/silk))
  * `vxlan-policy-agent` (from [silk-release](code.cloudfoundry.org/silk))
  * `policy-server`
  * `policy-server-internal`
  * `netmon` (from [silk-release](code.cloudfoundry.org/silk))

  The log lines for the following components will be not returned:
  * `iptables`(from [silk-release](code.cloudfoundry.org/silk)): We have limited room to add an identifier, and these logs are high-volume.
  * `garden-external-networker`: Garden will only print errors if creating a container fails.
  * `cni-wrapper-plugin`(from [silk-release](code.cloudfoundry.org/silk)): Only stdout and stderr are printed in garden logs.

* Container Create is Failing:

  If container create is failing check the garden logs, located on the cell VMs at `/var/vcap/sys/log/garden/garden.stdout.log`.
  Garden logs stdout and stderr from calls to the CNI plugin, you can find any errors related to the CNI ADD/DEL there. 

  Unsuccessful create will say things like `exit status 1` in the `stderr` field of the log message.

* Problems Creating Policies:

  Problems creating policies are usually related to issues on the policy server VM(s). Check the logs at `/var/vcap/sys/log/policy-server/policy-server.stdout.log`

  If a policy is successfully created you will see a log line created with the message `created-policies` along with other relevant data.
  If a policy is successfully deleted you will see a log line created with the message `deleted-policies` along with other relevant data.

### Enabling Debug Logging

The policy server log at the `info` level by default. The log level can be adjusted at runtime by making a request to the debug server running on the VM.
To enable debug logging ssh to the VM and make this request to the debug server:
```
curl -X POST -d 'DEBUG' localhost:31821/log-level
```
To switch back to info logging make this request:
```
curl -X POST -d 'INFO' localhost:31821/log-level
```
For the policy server, the debug server listens on port 31821 by default, it can be overridden by the manifest property `cf_networking.policy_server.debug_port`.

### Metrics

CF networking components emit metrics which can be consumed from the firehose, e.g. with the datadog firehose nozzle. Relevant metrics have theses prefixes:
-   `policy_server`


### Diagnosing and Recovering from Subnet Overlap

This section describes how to recover from a deploy of CF Networking with Silk which has an overlay network configured which conflicts with the entire CF subnet. We set `network` on the `silk-controller` to the same subnet as CF and BOSH (10.0.0.0/16). When we deploy we fail to bring up the first diego cell
  
```
17:31:56 | Updating instance diego-cell: diego-cell/4abb639b-33a9-4d8d-8a95-21c3863c7b0c (0) (canary) (00:03:25)
	    L Error: Timed out sending 'get_state' to baa8bc66-df64-4c2f-80d0-c090559ac28d after 45 seconds

17:35:22 | Error: Timed out sending 'get_state' to baa8bc66-df64-4c2f-80d0-c090559ac28d after 45 seconds
```

bosh vms shows:
```
$ bosh vms
Using environment 'https://104.196.19.37:25555' as client 'admin'

Task 44. Done

Deployment 'cf'

Instance                                          Process State       AZ  IPs          VM CID                                   VM Type
api/45e92d63-905b-4659-8400-3df97b97850b          running             z1  10.0.16.10   vm-bd55a96c-e4b4-4a33-53d0-7297e63d5bed  m3.large
api/f381c70a-58b7-4972-96d0-5a717122ce1e          running             z2  10.0.32.8    vm-3579ec81-6cad-4e44-42f7-170545c11dc2  m3.large
blobstore/a4c314e9-1b79-46bc-b572-c57e1be563b3    running             z1  10.0.16.9    vm-6a787749-9f7b-4fb7-7565-2f0b66ed1794  m3.large
cc-bridge/1999c3c2-7b4c-4ac0-8b43-189561edc7d6    running             z1  10.0.16.16   vm-ee138dd7-e01c-473b-402e-2a5896746e1f  m3.medium
cc-bridge/fb02bc5b-5b75-4e3b-8e1c-451b4baaafe6    running             z2  10.0.32.13   vm-5ddb0e3d-6c44-4bd0-5232-04c5d28e9151  m3.medium
cc-clock/d2a40e2f-b93f-4821-8d9e-b5e7f2fafec5     running             z1  10.0.16.15   vm-a94c4a98-ece2-4ad5-68f5-aa53f4c988c5  m3.large
cc-worker/ed3655bf-0d6a-4ea7-bd21-060d7c7146b7    running             z1  10.0.16.11   vm-f5f3f577-80fe-43b9-554f-7ea55f954c82  m3.medium
cc-worker/feb22fc9-a701-4dff-9c44-e920a03d8520    running             z2  10.0.32.9    vm-8465f4e3-d3c5-453a-7b88-0392e3efd160  m3.medium
consul/00d7d425-6291-4f8a-8313-7deae2fbd3c9       running             z2  10.0.32.4    vm-d61b76e9-45ae-432b-504d-f3290f7b09e6  m3.medium
consul/906080b6-28d2-411f-9a96-88d1556f0d82       running             z3  10.0.48.4    vm-0ba6eed9-20fa-4375-53ac-e6911fd60b49  m3.medium
consul/d2c6612c-cad6-412d-bac7-ac35c08895c5       running             z1  10.0.16.4    vm-a03136b0-e769-468a-6848-2ab8347782d1  m3.medium
diego-bbs/800ab3a6-529a-476d-ab07-95ee5847ad8f    running             z1  10.0.16.7    vm-c96eec32-b644-4c42-7e46-90a054eb6bbb  m3.large
diego-bbs/c99ce96a-1e4e-4e33-b972-3add40a831db    running             z2  10.0.32.6    vm-29d36c97-d372-4b7c-4cfd-ff58e4046107  m3.large
diego-brain/2a81d157-3e65-4335-b661-0393f867d252  running             z2  10.0.32.11   vm-fd18327e-5044-4056-4d23-c5c8a833c172  m3.medium
diego-brain/9811e507-4b3e-490a-839e-147f3eaa9089  running             z1  10.0.16.13   vm-04482d80-97cc-4eb8-44e2-0756f45529e1  m3.medium
diego-cell/460d0ead-a1b4-40c5-9f88-70327b536aa8   running             z2  10.0.32.12   vm-de76fbd8-cd3c-44ab-6ccc-bc0755909a91  r3.xlarge
diego-cell/4abb639b-33a9-4d8d-8a95-21c3863c7b0c   unresponsive agent  z1  10.0.16.14   vm-323a3988-5671-4b0d-4939-622d1886185f  r3.xlarge
doppler/0108ae24-b3ca-4d14-9cd6-c5640ffc359c      running             z1  10.0.16.17   vm-30645eea-47cd-428c-52a7-116ce26d46e9  m3.medium
doppler/05d36944-d4ee-4c40-9eba-f8abce03d942      running             z2  10.0.32.14   vm-97fc113f-d0b1-4bf4-6b2c-b99a82e88c74  m3.medium
etcd/2bef16f1-ce6e-4049-a3b9-728966135072         running             z1  10.0.16.5    vm-0b59d1db-bf0b-4a22-43ad-d4e04edad572  m3.medium
etcd/ca1a776d-405c-4687-800b-ea707d015abb         running             z3  10.0.48.5    vm-b671dcea-6235-4217-6f50-8172a414fd98  m3.medium
etcd/f7171ac1-d760-4e79-986f-52409f22008a         running             z2  10.0.32.5    vm-268b024e-00ee-4fe0-5dd7-5cd36a975661  m3.medium
log-api/3e8f0f29-1dfe-4620-ba26-6d51db8254ed      running             z1  10.0.16.18   vm-29a20458-fc53-426f-47ff-293394fbf416  t2.small
log-api/4bc34462-5ae8-4cc1-82d2-bc74ef2866f5      running             z2  10.0.32.15   vm-560e68ef-0629-4457-41b4-fbc76b2d09e6  t2.small
mysql/e448ba11-5f14-46f2-876c-e0e926fef2d5        running             z1  10.0.16.6    vm-3be727a8-da1c-427a-6701-16ee19be6264  m3.large
nats/5fd2bb98-f384-4bbf-86b9-b5e888543b1a         running             z2  10.0.47.191  vm-5d271164-c9db-44d8-6dd2-a014ff075eaa  c3.large
nats/b977aa7e-ced3-4e68-8dd9-bd5bb5640dec         running             z1  10.0.31.191  vm-86297c9b-19b5-45db-5dbc-dd3ed5638d42  c3.large
router/40f14eb0-47f2-4b1e-831d-6f1c9620c5ef       running             z3  10.0.48.6    vm-8590c4d2-2e89-4d72-7841-6d191d36a31a  m3.medium
router/7a932987-5a50-4a2e-a221-c9ad1d601e6f       running             z2  10.0.32.10   vm-91df6232-7572-479f-51d8-5d0b24f8cdc3  m3.medium
router/a947e6e0-d814-434f-a66b-2b610e9eae93       running             z1  10.0.16.12   vm-afeed179-d644-441b-7f49-add45d73b905  m3.medium
uaa/15d93dba-7b04-4d25-a244-7d1a93712e47          running             z1  10.0.16.8    vm-715d445e-117d-4fd3-710a-7e92b028ec1b  m3.medium
uaa/4c334724-7362-47b6-917e-3502566a8fad          running             z2  10.0.32.7    vm-e8cb1c9a-02c8-4b2b-7c9c-21261cb5e5a0  m3.medium
```

Trying to roll back fails since the deployment lock is still being held.

These commands get the deployment unstuck, so the operator can roll back to a previous version or correct the configuration:
```
bosh update-resurrection off
bosh ignore diego-cell/4abb639b-33a9-4d8d-8a95-21c3863c7b0c
bosh delete-vm vm-a4f6c259-c1f8-4b35-746e-f1db3359fad6
```

Once the deploy is complete run:
```
bosh update-resurrection on
bosh unignore diego-cell/4abb639b-33a9-4d8d-8a95-21c3863c7b0c
```


### Inspecting VTEP configuration
The VXLAN tunnel endpoint can be inspected using the `ip` utility from the `iproute2` package.

From the Diego cell, install a recent version of `iproute2` and its dependency `libmnl`:
```
curl -o /tmp/iproute2.deb -L http://mirrors.kernel.org/ubuntu/pool/main/i/iproute2/iproute2_4.3.0-1ubuntu3_amd64.deb
curl -o /tmp/libmnl0.deb -L http://mirrors.kernel.org/ubuntu/pool/main/libm/libmnl/libmnl0_1.0.3-5_amd64.deb
dpkg -i /tmp/*.deb
```

Then you can see details of the VTEP device by running
```
ip -d link list silk-vtep
```
which should resemble
```
3: silk-vtep: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1410 qdisc noqueue state UNKNOWN mode DEFAULT group default
    link/ether ee:ee:0a:ff:17:00 brd ff:ff:ff:ff:ff:ff promiscuity 0
    vxlan id 1 local 10.0.32.15 dev eth0 srcport 0 0 dstport 4789 nolearning ageing 300 gbp
```

Note `srcport 0 0 dstport 4789`.  The `dstport PORT` value is the external UDP port used by the VTEP for all encapsulated VXLAN packets.
See [the `vtep_port` on the `silk-daemon` job](https://github.com/cloudfoundry/silk-release/blob/a68ed558438405dabba56297e787dfbebd34f080/jobs/silk-daemon/spec#L22-L24).

For more details, look at `man ip-link`.


### When packets won't flow (with Silk)
If you've installed `cf-networking-release` and `silk-release` and find that you don't have connectivity between app containers,
there are some basic troubleshooting steps to follow:

First, verify basic infrastructure network connectivity between diego cells.  You can see the infrastructure ip from running `bosh vms`.
`bosh ssh diego-cell/0` and `bosh ssh diego-cell/1`.  Then `ping` the infrastructure (underlay) IP of one cell from the other cell.

If that part works, the next would be to test the overlay network connectivity between diego cells themselves.
Each diego cell has an overlay IP address for itself.  You can discover this by running
```
ip addr show silk-vtep
```
from the cell.  It will show something like:
```
294: silk-vtep: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1410 qdisc noqueue state UNKNOWN group default
    link/ether ee:ee:0a:ff:22:00 brd ff:ff:ff:ff:ff:ff
    inet 10.255.34.0/16 scope link silk-vtep
       valid_lft forever preferred_lft forever
```
Then ping that overlap IP address from a different cell
```
ping 10.255.34.0
```
This step should always succeed.

If it doesn't, use `tcpdump` on each side of the connection to inspect packets.

First, [use a recent version of the `ip` utility to discover your VTEP port](#inspecting-vtep-configuration), or confirm that your `vtep_port` is set to `4789`.

Then, from each diego cell, open a terminal and run
```
tcpdump -n port 4789
```

In a separate terminal, re-run the `ping $CELL_OVERLAY_ADDRESS`.  The `tcpdump` output on each cell should show a single request/reply packet pair:

```
02:19:44.965688 IP 10.0.16.17.43146 > 10.0.32.15.4789: VXLAN, flags [I] (0x08), vni 1
IP 10.255.34.0 > 10.255.23.0: ICMP echo request, id 56571, seq 1, length 64
02:19:44.966850 IP 10.0.32.15.41667 > 10.0.16.17.4789: VXLAN, flags [I] (0x08), vni 1
IP 10.255.23.0 > 10.255.34.0: ICMP echo reply, id 56571, seq 1, length 64
```

Note that each packet shows up as two lines in tcpdump:
- the first line is the underlay network packet, destined for VXLAN listener on UDP port 4789.
- the second line shows the encapsulated, overlay ICMP packet

If `tcpdump` shows outgoing packets from the sending cell, but no incoming packets on the receiving cell, your infrastructure may be blocking access to the VXLAN port `4789`.
Consider changing the `vtep_port` value in your BOSH manifest and re-deploying.


### Debugging C2C Packets

To determine a failure on a c2c packet, `bosh ssh` onto a cell suspected of hosting an app that is not receiving failing packets.

As above, set up a packet capture
```
tcpdump -n -XX port 4789
```
where `4789` is the `vtep_port` setting on the `silk-daemon` job (or [use a recent version of the `ip` utility to discover your VTEP port](#inspecting-vtep-configuration)).

Here's a packet capture from an HTTP request over the overlay network
```
03:39:07.934028 IP 10.0.32.15.41669 > 10.0.16.17.4789: VXLAN, flags [I] (0x88), vni 1
IP 10.255.82.67.58054 > 10.255.144.63.8080: Flags [P.], seq 1:83, ack 1, win 215, options [nop,nop,TS val 9490690 ecr 9562410], length 82: HTTP: GET / HTTP/1.1
        0x0000:  4201 0a00 0001 4201 0a00 200f 0800 4500  B.....B.......E.
        0x0010:  00b8 69c1 0000 4011 cc54 0a00 200f 0a00  ..i...@..T......
        0x0020:  1011 a2c5 12b5 00a4 0000 8800 0003 0000  ................
        0x0030:  0100 eeee 0aff 9000 eeee 0aff 5200 0800  ............R...
        0x0040:  4500 0086 ae2d 4000 3f06 94c4 0aff 5243  E....-@.?.....RC
        0x0050:  0aff 903f e2c6 1f90 84e6 7f2b 872d 297f  ...?.......+.-).
        0x0060:  8018 00d7 1be5 0000 0101 080a 0090 d102  ................
        0x0070:  0091 e92a 4745 5420 2f20 4854 5450 2f31  ...*GET./.HTTP/1
        0x0080:  2e31 0d0a 5573 6572 2d41 6765 6e74 3a20  .1..User-Agent:.
        0x0090:  6375 726c 2f37 2e33 352e 300d 0a48 6f73  curl/7.35.0..Hos
        0x00a0:  743a 2031 302e 3235 352e 3134 342e 3633  t:.10.255.144.63
        0x00b0:  3a38 3038 300d 0a41 6363 6570 743a 202a  :8080..Accept:.*
        0x00c0:  2f2a 0d0a 0d0a                           /*....
```
Note:
- `vni 1` corresponds to the 3 bytes starting at `0x002E`
- The VXLAN port `4789` in decimal is `12b5` in hex.  You can see that at address `0x0024`
- In this example, the sending application was tagged with [VXLAN Group Based Policy](https://tools.ietf.org/html/draft-smith-vxlan-group-policy-04#section-2.1) (GBP)
  ID `0x0003`.  This is at address `0x002C`
- The overlay destination ip address of `10.255.144.63` is `0aff 903f`.  You can see that at address `0x0050`
- The overlay destination port `8080` in decimal is `1f90` in hex.  You can see that at address `0x0066`

To see which application is assigned the GBP tag `0x0003`, query the Network Policy Server:

```
cf curl /networking/v1/external/tags | jq .
```

```json
{
  "tags": [
    {
      "id": "f0782c6c-a2fe-4cad-a206-3970c57bb532",
      "tag": "0001"
    },
    {
      "id": "0f2e3339-0d19-4e87-b381-030c3619bd6b",
      "tag": "0002"
    },
    {
      "id": "1ee630fd-528a-4d80-97cd-dea170be759d",
      "tag": "0003"
    }
  ]
}
```
The `id` associated with tag `0003` is the source application guid, the result of `cf app --guid app-name`.

You can also inspect `/var/vcap/data/container-metadata/store.json` on a diego cell to see the IP address and CF app metadata for every container:
```json
{
   "53069156-e8cf-4ca5-5a0a-838e" : {
      "handle" : "53069156-e8cf-4ca5-5a0a-838e",
      "ip" : "10.255.82.67",
      "metadata" : {
         "app_id" : "1ee630fd-528a-4d80-97cd-dea170be759d",
         "space_id" : "49dae64f-8231-4294-9882-1490c120f839",
         "org_id" : "cfe7a279-ad29-48d4-b188-813e85d2b621",
         "policy_group_id" : "1ee630fd-528a-4d80-97cd-dea170be759d"
      }
   }
}
```
Note that:
- the `ip` in the above metadata matches the overlay source IP from the `tcpdump` output
- the `app_id` in the above metadata matches tag `0003` in the `tags` result
- the `tag` of `0003` can be read off the `tcpdump` output

### Debugging Non-C2C Packets

To capture non-c2c packets (destination is an external address), run the following:

```
tcpdump -v -XX -i any
```

The packets that you are interested in will be packets with the source ip being in the container network.

To filter by destination address, you may add `-n dst host <destination-ip>` to the `tcpdump` command. For example:

```
tcpdump -v -XX -i any -n dst host 96.126.115.72
```

Once you have a packet and want to find information about the application,
find the assigned container ip in the packet header. For the example below, the ip is 10.255.29.3.
```
18:49:04.749158 IP (tos 0x0, ttl 64, id 25116, offset 0, flags [DF], proto TCP (6), length 114)
  10.255.29.3.34638 > lax17s05-in-f14.1e100.net.http: Flags [P.], cksum 0xda6f (incorrect -> 0x5709), seq 1:75, ack 1, win 28200, length 74: HTTP, length: 74
      GET / HTTP/1.1
      User-Agent: curl/7.35.0
      Host: google.com
      Accept: */*

      0x0000:  0a58 0aff 1d01 0a58 0aff 1d03 0800 4500  .X.....X......E.
      0x0010:  0072 621c 4000 4006 fe5e 0aff 1d03 d83a  .rb.@.@..^.....:
      0x0020:  d9ce 874e 0050 3a0f 5211 3aa9 9402 5018  ...N.P:.R.:...P.
      0x0030:  6e28 da6f 0000 4745 5420 2f20 4854 5450  n(.o..GET./.HTTP
      0x0040:  2f31 2e31 0d0a 5573 6572 2d41 6765 6e74  /1.1..User-Agent
      0x0050:  3a20 6375 726c 2f37 2e33 352e 300d 0a48  :.curl/7.35.0..H
      0x0060:  6f73 743a 2067 6f6f 676c 652e 636f 6d0d  ost:.google.com.
      0x0070:  0a41 6363 6570 743a 202a 2f2a 0d0a 0d0a  .Accept:.*/*....
```

On the same cell which the packet was captured, run
```
less /var/vcap/data/container-metadata/store.json | json_pp
```
and find the entry with the ip. If no entry exists, check the `store.json` on other cells.

The associated `app_id` is the application guid.

Example of `store.json` output:
```
{
 "9bce657c-b92f-422b-60e0-227a66ad8b48" : {
    "metadata" : {
       "space_id" : "601577f3-7c2d-4d98-8029-0bd03b6a0682",
       "app_id" : "aa4117a2-5e34-4648-9d42-8260380267cc",
       "policy_group_id" : "aa4117a2-5e34-4648-9d42-8260380267cc",
       "org_id" : "ff585363-1164-49b2-bbf3-55dd0cb06597"
    },
    "ip" : "10.255.29.4",
    "handle" : "9bce657c-b92f-422b-60e0-227a66ad8b48"
 },
 "cf760bdc-ebf9-414e-4a88-29dc8820643e" : {
    "ip" : "10.255.29.3",
    "metadata" : {
       "policy_group_id" : "f028b20b-7203-4743-ab96-da2bf05fae45",
       "space_id" : "601577f3-7c2d-4d98-8029-0bd03b6a0682",
       "app_id" : "f028b20b-7203-4743-ab96-da2bf05fae45",
       "org_id" : "ff585363-1164-49b2-bbf3-55dd0cb06597"
    },
    "handle" : "cf760bdc-ebf9-414e-4a88-29dc8820643e"
 }
}
```
