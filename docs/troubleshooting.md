# Troubleshooting

NOTE: If you are having problems, first consult our [known issues doc](known-issues.md).

### Checking Logs

  0. Container Create is Failing:

  If container create is failing check the garden logs, located on the cell VMs at `/var/vcap/sys/log/garden/garden.stdout.log`.
  Garden logs stdout and stderr from calls to the CNI plugin, you can find any errors related to the CNI ADD/DEL there. 

  Unsuccessful create will say things like `exit status 1` in the `stderr` field of the log message.

  0. Problems Creating Policies:

  Problems creating policies are usually related to issues on the policy server VM(s). Check the logs at `/var/vcap/sys/log/policy-server/policy-server.stdout.log`

### Enabling Debug Logging

  The policy server and VXLAN policy agent log at the `info` level by default. The log level can be adjusted at runtime by making a request to the debug server running on the VM.
  To enable debug logging ssh to the VM and make this request to the debug server:
  ```
  curl -X POST -d 'DEBUG' localhost:22222/log-level
  ```
  To switch back to info logging make this request:
  ```
  curl -X POST -d 'INFO' localhost:22222/log-level
  ```
  The debug server listens on port 22222 by default, it can be overridden by the manifest properties `policy-server.debug_server_port` and `vxlan-policy-agent.debug_server_port`

### Enabling IPTables Logging for Container to Container Traffic

  Logging for policy iptables rules can be enabled through the VXLAN policy agent debug server. SSH to a cell VM and make this request to enable logging on the VM:
  ```
  curl -X PUT -d '{"enabled": true}' localhost:22222/iptables-c2c-logging
  ```
  To disable:
  ```
  curl -X PUT -d '{"enabled": false}' localhost:22222/iptables-c2c-logging
  ```

  This can be configured at startup via the
  `cf_networking.iptables_logging` property. It defaults
  to `false`. This property is used by the `vxlan-policy-agent` and the `silk-cni` jobs.

  Logs from iptables end up in `/var/log/kern.log`.

  Example of a rejected connection:
  ```
  May  3 23:34:07 localhost kernel: [87921.493829] DENY_C2C_cb40f81e-52ce-41c5- IN=s-010255015007 OUT=s-010255015013 MAC=aa:aa:0a:ff:0f:07:ee:ee:0a:ff:0f:07:08:00 SRC=10.255.15.7 DST=10.255.15.13 LEN=60 TOS=0x00 PREC=0x00 TTL=63 ID=35889 DF PROTO=TCP SPT=36004 DPT=723 WINDOW=29200 RES=0x00 SYN URGP=0 MARK=0x2
  ```

  Example of an accepted connection, note that the prefix `OK_0003` indicates the packet with tag 3 was accepted:
  ```
  May  3 23:35:07 localhost kernel: [87981.320056] OK_0002_e9e8959f-3828-4136-8 IN=s-010255015007 OUT=s-010255015013 MAC=aa:aa:0a:ff:0f:07:ee:ee:0a:ff:0f:07:08:00 SRC=10.255.15.7 DST=10.255.15.13 LEN=52 TOS=0x00 PREC=0x00 TTL=63 ID=43997 DF PROTO=TCP SPT=60012 DPT=8080 WINDOW=237 RES=0x00 ACK URGP=0 MARK=0x2
  ```

### Enabling IPTables Logging for ASG Traffic

  Logging for ASG iptables rules can be configured at startup via the
  `cf_networking.iptables_logging` property. It defaults
  to `false`.

  Logs from iptables end up in `/var/log/kern.log`.

  Example of a rejected connection, note that the prefix `DENY_b6de7d0c-4792-4614-5e51-` indicates that an app instance with instance guid starting with `b6de7d0c-4792-4614-5e51-` was not able to connect to `10.0.16.8`:

  ```
  May  3 23:35:58 localhost kernel: [88032.025828] DENY_d538d169-f2f6-4587-77b1 IN=s-010255015007 OUT=eth0 MAC=aa:aa:0a:ff:0f:07:ee:ee:0a:ff:0f:07:08:00 SRC=10.255.15.7 DST=10.10.10.1 LEN=60 TOS=0x00 PREC=0x00 TTL=63 ID=61375 DF PROTO=TCP SPT=49466 DPT=80 WINDOW=29200 RES=0x00 SYN URGP=0 MARK=0x2
  ```

  Example of an accepted connection, note that the prefix `OK_b6de7d0c-4792-4614-5e51-4c` indicates that an app instance with an instance guid starting with `b6de7d0c-4792-4614-5e51-4c` was able to connect to `93.184.216.34`:
  ```
  May  3 23:35:35 localhost kernel: [88008.920287] OK_d538d169-f2f6-4587-77b1-f IN=s-010255015007 OUT=eth0 MAC=aa:aa:0a:ff:0f:07:ee:ee:0a:ff:0f:07:08:00 SRC=10.255.15.7 DST=173.194.210.139 LEN=60 TOS=0x00 PREC=0x00 TTL=63 ID=45400 DF PROTO=TCP SPT=35236 DPT=80 WINDOW=29200 RES=0x00 SYN URGP=0 MARK=0x2
  ```

### Metrics

  CF networking components emit metrics which can be consumed from the firehose, e.g. with the datadog firehose nozzle. Relevant metrics have theses prefixes:
  -   `netmon`
  -   `vxlan_policy_agent`
  -   `policy_server`


### Debugging C2C Packets

  To determine a failure on a c2c packet, `bosh ssh` onto a cell suspected of hosting an app that is not receiving failing packets.

  To find relevant packets, run the following command
  ```
  tcpdump -T vxlan -v -XX -i <interface>
  ```
  where `<interface>` is the lowest level BROADCAST address found from running
  ```
  ip link
  ```
  For the example output of this command below, interface is `eth0`.
  ```
  1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN mode DEFAULT group default qlen 1
      link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
  2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1460 qdisc mq state UP mode DEFAULT group default qlen 1000
      link/ether 42:01:0a:00:10:0e brd ff:ff:ff:ff:ff:ff
  349: silk-vtep: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1410 qdisc noqueue state UNKNOWN mode DEFAULT group default
      link/ether ee:ee:0a:ff:0f:00 brd ff:ff:ff:ff:ff:ff
  353: s-010255015002@if352: <BROADCAST,MULTICAST,NOARP,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP mode DEFAULT group default
      link/ether aa:aa:0a:ff:0f:02 brd ff:ff:ff:ff:ff:ff
  ```

  If packet capture is already set up, a packet is distinguished as VXLAN in the packet header.

  C2C packets should contain `vni 1` in the header, similar to
  ```
  3:26:56.211447 IP (tos 0x0, ttl 64, id 60857, offset 0, flags [none], proto UDP (17), length 110)
    cell-z1-0.node.dc1.cf.internal.57703 > cell-z1-1.node.dc1.cf.internal.8472: VXLAN, flags [I] (0x88), vni 1
  ```

  We can now use to VXLAN packet to find information about the app that sent it.

  First, find the GBP tag in the body of the packet.
  Information concerning the GBP tag will be 4 bytes after the VXLAN port, which is always `0x2118`.

  In the example below, the relevant information is `0x8800 0001`.
  The first two bytes (`0x88`) tell us that a GBP tag exists.
  If a GBP tag does not exist, the two bytes will be `0x08`.

  ```
  0x0000:  0e4e b0d2 3b72 96de b050 aa1a 0800 4500  .N..;r...P....E.
	0x0010:  006e edb9 0000 4011 56d1 0af4 1004 0af4  .n....@.V.......
	0x0020:  1009 e167 2118 005a 0000 8800 0001 0000  ...g!..Z........
	0x0030:  0100 32fe ee1f 5bc6 dec2 c712 ac75 0800  ..2...[......u..
	0x0040:  4500 003c 18dd 4000 3f06 80d4 0aff 4003  E..<..@.?.....@.
	0x0050:  0aff 4c0a dd62 1f90 4cc9 3652 0000 0000  ..L..b..L.6R....
	0x0060:  a002 6e28 a239 0000 0204 0582 0402 080a  ..n(.9..........
	0x0070:  0071 2e8f 0000 0000 0103 0307            .q..........
  ```

  If there is a tag, the next six bytes represent the tag. For the same example, the tag is `0x000001`.

  With a GBP tag, app information can be found by running
  ```
  cf curl /networking/v0/external/tags
  ```
  and parsing the output based on the GBP tag. The associated `id` is the application guid.

  Example of `cf curl` output:
  ```
  {
  "tags": [
    {
      "id": "f028b20b-7203-4743-ab96-da2bf05fae45",
      "tag": "0001"
    },
    {
      "id": "aa4117a2-5e34-4648-9d42-8260380267cc",
      "tag": "0002"
    }
  ]
  }
  ```

  If the VXLAN packet does not have a GBP tag, continue to examine the packet.

  Twenty bytes before the VXLAN port is the eight byte hex representation of the source ip for the cell that hosts the application.
  In the example above, this value is `0x0af41004`.

  Thirty bytes after the GBP tag (forty-six bytes after the VXLAN port) is the eight byte hex representation of the source ip for the application.
  In the example above, this value is `0x0aff4c0a`.

  Now `bosh ssh` onto the cell with the source ip just found. Run
  ```
  less /var/vcap/data/container-metadata/store.json
  ```
  and search the resulting output for an entry with the same ip as the application source ip found.
  The relevant application guid can be found in the associated metadata as `app_id`.

  Example of `store.json` output:
  ```
  {
   "6d2131bb-fe5e-47d7-7e46-d925cf6db115" : {
      "ip" : "10.255.64.3",
      "metadata" : {
         "policy_group_id" : "dc947050-d073-4a8f-8693-be10a1ae8553",
         "org_id" : "98ad4a19-2a9f-412e-9431-54b0f9dfede1",
         "space_id" : "756e1478-3a30-48a5-a308-93aaa1dd178f",
         "app_id" : "dc947050-d073-4a8f-8693-be10a1ae8553"
      },
      "handle" : "6d2131bb-fe5e-47d7-7e46-d925cf6db115"
   }
  }
  ```

### Debugging Non-C2C Packets

  If you have a packet that is not c2c (destination is an external address), and want to find information about the application,
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
