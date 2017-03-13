# Troubleshooting

NOTE: If you are having problems, first consult our [known issues doc](known-issues.md).

### Checking Logs

  0. Container Create is Failing:

  If container create is failing check the garden logs, located on the cell VMs at `/var/vcap/sys/log/garden/garden.stdout.log`.
  Garden logs stdout and stderr from calls to the CNI plugin, you can find any errors related to the CNI ADD/DEL there. 
  An example of a successful container create:

  ```
  {
    "timestamp": "1485210024.178133965",
    "source": "guardian",
    "message": "guardian.create.external-networker-result",
    "log_level": 0,
    "data": {
      "action": "up",
      "handle": "executor-healthcheck-e55d1832-a59c-47c8-780c-5ed4056089f3",
      "session": "555",
      "stderr": "{\"timestamp\":\"1485210024.090760708\",\"source\":\"container-networking.garden-external-networker\",\"message\":\"container-networking.garden-external-networker.action\",\"log_level\":1,\"data\":{\"action\":\"up\"}}\n{\"timestamp\":\"1485210024.091046810\",\"source\":\"container-networking.garden-external-networker\",\"message\":\"container-networking.garden-external-networker.loaded-config\",\"log_level\":1,\"data\":{\"network\":{\"cniVersion\":\"0.2.0\",\"name\":\"cni-wrapper\",\"type\":\"cni-wrapper-plugin\",\"ipam\":{},\"dns\":{}},\"raw\":\"{\\n  \\\"name\\\": \\\"cni-wrapper\\\",\\n  \\\"type\\\": \\\"cni-wrapper-plugin\\\",\\n  \\\"cniVersion\\\": \\\"0.2.0\\\",\\n  \\\"datastore\\\": \\\"/var/vcap/data/container-metadata/store.json\\\",\\n  \\\"iptables_lock_file\\\": \\\"/var/vcap/data/garden-cni/iptables.lock\\\",\\n  \\\"overlay_network\\\": \\\"10.255.0.0/16\\\",\\n  \\\"delegate\\\": {\\n    \\\"name\\\": \\\"cni-flannel\\\",\\n    \\\"type\\\": \\\"flannel\\\",\\n    \\\"subnetFile\\\": \\\"/var/vcap/data/flannel/subnet.env\\\",\\n    \\\"dataDir\\\": \\\"/var/vcap/data/flannel/data\\\",\\n    \\\"delegate\\\": {\\n      \\\"bridge\\\": \\\"cni-flannel0\\\",\\n      \\\"isDefaultGateway\\\": true,\\n      \\\"ipMasq\\\": false\\n     }\\n  }\\n}\\n\"}}\n{\"timestamp\":\"1485210024.094831467\",\"source\":\"container-networking.garden-external-networker\",\"message\":\"container-networking.garden-external-networker.up-add-network-start\",\"log_level\":1,\"data\":{\"networkConfig\":\"{\\n  \\\"name\\\": \\\"cni-wrapper\\\",\\n  \\\"type\\\": \\\"cni-wrapper-plugin\\\",\\n  \\\"cniVersion\\\": \\\"0.2.0\\\",\\n  \\\"datastore\\\": \\\"/var/vcap/data/container-metadata/store.json\\\",\\n  \\\"iptables_lock_file\\\": \\\"/var/vcap/data/garden-cni/iptables.lock\\\",\\n  \\\"overlay_network\\\": \\\"10.255.0.0/16\\\",\\n  \\\"delegate\\\": {\\n    \\\"name\\\": \\\"cni-flannel\\\",\\n    \\\"type\\\": \\\"flannel\\\",\\n    \\\"subnetFile\\\": \\\"/var/vcap/data/flannel/subnet.env\\\",\\n    \\\"dataDir\\\": \\\"/var/vcap/data/flannel/data\\\",\\n    \\\"delegate\\\": {\\n      \\\"bridge\\\": \\\"cni-flannel0\\\",\\n      \\\"isDefaultGateway\\\": true,\\n      \\\"ipMasq\\\": false\\n     }\\n  }\\n}\\n\",\"runtimeConfig\":{\"ContainerID\":\"executor-healthcheck-e55d1832-a59c-47c8-780c-5ed4056089f3\",\"NetNS\":\"/var/vcap/data/garden-cni/container-netns/executor-healthcheck-e55d1832-a59c-47c8-780c-5ed4056089f3\",\"IfName\":\"eth0\",\"Args\":null}}}\n{\"timestamp\":\"1485210024.152931213\",\"source\":\"container-networking.garden-external-networker\",\"message\":\"container-networking.garden-external-networker.up-add-network-result\",\"log_level\":1,\"data\":{\"name\":\"cni-wrapper\",\"result\":\"IP4:{IP:{IP:10.255.67.13 Mask:ffffff00} Gateway:10.255.67.1 Routes:[{Dst:{IP:10.255.0.0 Mask:ffff0000} GW:\\u003cnil\\u003e} {Dst:{IP:0.0.0.0 Mask:00000000} GW:10.255.67.1}]}, DNS:{Nameservers:[] Domain: Search:[] Options:[]}\",\"type\":\"cni-wrapper-plugin\"}}\n{\"timestamp\":\"1485210024.153006077\",\"source\":\"container-networking.garden-external-networker\",\"message\":\"container-networking.garden-external-networker.up-complete\",\"log_level\":1,\"data\":{\"numConfigs\":1}}\n",
      "stdin": "{\"Pid\":19335,\"Properties\":{}}",
      "stdout": "{\"properties\":{\"garden.network.container-ip\":\"10.255.67.13\",\"garden.network.host-ip\":\"255.255.255.255\"}}\n"
    }
  }
  ```

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
  `cf_networking.vxlan_policy_agent.iptables_c2c_logging` property. It defaults
  to `false`.

  Logs from iptables end up in `/var/log/kern.log`.

  Example of a rejected connection:
  ```
  Jan 23 23:15:14 localhost kernel: [856287.885695] REJECT_REMOTE:IN=flannel.1 OUT=cni-flannel0 MAC=f6:c9:e6:4e:23:5c:b6:76:98:0e:64:0c:08:00 SRC=10.255.69.132 DST=10.255.31.137 LEN=60 TOS=0x00 PREC=0x00 TTL=62 ID=8033 DF PROTO=TCP SPT=33254 DPT=7000 WINDOW=26733 RES=0x00 SYN URGP=0
  ```

  Example of an accepted connection, note that the prefix `OK_0003` indicates the packet with tag 3 was accepted:
  ```
  Jan 23 23:15:38 localhost kernel: [856311.500733] OK_0003_9edc60d3-6cc8-4dc4-82IN=flannel.1 OUT=cni-flannel0 MAC=f6:c9:e6:4e:23:5c:b6:76:98:0e:64:0c:08:00 SRC=10.255.69.132 DST=10.255.31.137 LEN=60 TOS=0x00 PREC=0x00 TTL=62 ID=9292 DF PROTO=TCP SPT=37042 DPT=8080 WINDOW=26733 RES=0x00 SYN URGP=0 MARK=0x3
  ```

### Metrics

  CF networking components emit metrics which can be consumed from the firehose, e.g. with the datadog firehose nozzle. Relevant metrics have theses prefixes:
  -   `netmon`
  -   `vxlan_policy_agent`
  -   `policy_server`
