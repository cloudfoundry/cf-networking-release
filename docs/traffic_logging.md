# Traffic logging

## Enabling kernel logging

* When `cf_networking.iptables_logging` is set to `true` on the
`vxlan_policy_agent` job, C2C iptables logs will be written to
`/var/log/kern.log`.
* When `cf_networking.iptables_logging` is set to `true` on
the `silk-cni` job, ASG iptables logs will be written to
`/var/log/kern.log`.

## Enabling augmented logging
In addition to the above steps to enable kernel logging, when the `iptables-logger` job from the
`cf-networking` release is added to your `diego-cell`, augmented traffic logs
(logs with the app/space/org info) will be written to
`/var/vcap/sys/log/iptables-logger/iptables.log`.


## Forwarding logs to an external syslog server

Deploy [syslog-release](https://github.com/cloudfoundry/syslog-release) to forward your logs to an external syslog server.

To avoid forwarding both the kernel logs and the augmented logs from the `iptables-logger` job:
* Set [`syslog.custom_rule`](https://bosh.io/jobs/syslog_forwarder?source=github.com/cloudfoundry/syslog-release&version=11#p=syslog.custom_rule) on the `syslog_forwarder` job to `if $programname == 'kernel' then ~` 
* Set [`syslog.blackbox.source_dir`](https://bosh.io/jobs/syslog_forwarder?source=github.com/cloudfoundry/syslog-release&version=11#p=syslog.blackbox.source_dir) on the `syslog_forwarder` job to `/var/vcap/sys/log`

Doing so will ignore logs in `/var/log/kern.log` but will still forward the augmented logs produced by `iptables-logger`.

## Log Volume and Performance

In [our tests](https://docs.google.com/document/d/1LufBEE94d2FulPwxaP-JxeFTCxGiV68MVQV1wJmJxOc/edit), turning on IP tables logging
had no significant impact on system performance.

From our investigation, it appears that CPU capabilities are the bottleneck for application performance. Request failures start to occur at around the same request rate regardless of whether logging is enabled.

Each iptables log line is approximately 630 bytes. This means 4000 log lines is roughly 2.5 MB.

For example, a cell with applications that are collectively receiving 2000 requests/second will generate
approximately 2.5 MB/second in logs, assuming each request creates two log lines (1 for DNS lookup and 1 for the actual request).

## Rate Limiting

### Denied logs
Denied logs are rate-limited using the limit module of iptables. Each packet produces a log line until the rate limit for a given source/destination is reached. This rate limit is configured by `cf_networking.iptables_denied_logs_per_sec` on the `silk-cni` job.

### Accepted logs
Accepted logs use the conntrack module of iptables. A single log line exists per connection.

The exception is logs for the UDP protocol, which are rate-limited using the limit module, similar to deny logs. The rate limit is configured by `iptables_accepted_udp_logs_per_sec` on the `silk-cni` and `vxlan-policy-agent` jobs.

## Sample outputs
### ASG allowed

Kernel log:
```
Jul 24 19:19:10 localhost kernel: [1468637.581122] OK_bfce786c-ab07-40ad-79f9-8
IN=s-010255073003 OUT=eth0 MAC=aa:aa:0a:ff:49:03:ee:ee:0a:ff:49:03:08:00
SRC=10.255.73.3 DST=8.8.8.8 LEN=60 TOS=0x00 PREC=0x00 TTL=63 ID=15946 DF
PROTO=TCP SPT=51858 DPT=80 WINDOW=27400 RES=0x00 SYN URGP=0 MARK=0x1
```

`iptables-logger` log:
```
{
  "timestamp": "1500923950.331232071",
  "source": "cfnetworking.iptables",
  "message": "cfnetworking.iptables.egress-allowed",
  "log_level": 1,
  "data": {
    "packet": {
      "direction": "egress",
      "allowed": true,
      "src_ip": "10.255.73.3",
      "dst_ip": "8.8.8.8",
      "src_port": 51858,
      "dst_port": 80,
      "protocol": "TCP",
      "mark": "0x1",
      "icmp_type": 0,
      "icmp_code": 0
    },
    "source": {
      "container_id": "bfce786c-ab07-40ad-79f9-8f21",
      "app_guid": "bc6f229d-5e4a-4c41-a63f-e8795496c283",
      "space_guid": "b9f86312-a7d7-4bcf-b70f-a440436c210b",
      "organization_guid": "604bd59e-4139-4734-a3be-4e97836eb790",
      "host_ip": "10.0.16.15",
      "host_guid": "0455ec2b-11fa-41ab-9d1c-f3a575cd55ea"
    }
  }
}
```

### ASG denied

Kernel log:
```
Jul 24 19:15:12 localhost kernel: [1468399.963562] DENY_bfce786c-ab07-40ad-79f9
IN=s-010255073003 OUT=eth0 MAC=aa:aa:0a:ff:49:03:ee:ee:0a:ff:49:03:08:00
SRC=10.255.73.3 DST=10.10.10.10 LEN=60 TOS=0x00 PREC=0x00 TTL=63 ID=51140 DF
PROTO=TCP SPT=36296 DPT=80 WINDOW=27400 RES=0x00 SYN URGP=0 MARK=0x1
```

`iptables-logger` log:
```
{
  "timestamp": "1500923712.879277706",
  "source": "cfnetworking.iptables",
  "message": "cfnetworking.iptables.egress-denied",
  "log_level": 1,
  "data": {
    "packet": {
      "direction": "egress",
      "allowed": false,
      "src_ip": "10.255.73.3",
      "dst_ip": "10.10.10.10",
      "src_port": 36296,
      "dst_port": 80,
      "protocol": "TCP",
      "mark": "0x1",
      "icmp_type": 0,
      "icmp_code": 0
    },
    "source": {
      "container_id": "bfce786c-ab07-40ad-79f9-8f21",
      "app_guid": "bc6f229d-5e4a-4c41-a63f-e8795496c283",
      "space_guid": "b9f86312-a7d7-4bcf-b70f-a440436c210b",
      "organization_guid": "604bd59e-4139-4734-a3be-4e97836eb790",
      "host_ip": "10.0.16.15",
      "host_guid": "0455ec2b-11fa-41ab-9d1c-f3a575cd55ea"
    }
  }
}
```

### c2c allowed

Kernel log:
```
Jul 24 19:21:10 localhost kernel: [1468757.382151] OK_0001_bc6f229d-5e4a-4c41-a
IN=s-010255073003 OUT=s-010255073002
MAC=aa:aa:0a:ff:49:03:ee:ee:0a:ff:49:03:08:00 SRC=10.255.73.3 DST=10.255.73.2
LEN=60 TOS=0x00 PREC=0x00 TTL=63 ID=14751 DF PROTO=TCP SPT=46936 DPT=8080
WINDOW=27400 RES=0x00 SYN URGP=0 MARK=0x1
```

`iptables-logger` log:
```
{
  "timestamp": "1500924070.182554722",
  "source": "cfnetworking.iptables",
  "message": "cfnetworking.iptables.ingress-allowed",
  "log_level": 1,
  "data": {
    "destination": {
      "container_id": "d5978989-1401-49ff-46cd-33e5",
      "app_guid": "bc6f229d-5e4a-4c41-a63f-e8795496c283",
      "space_guid": "b9f86312-a7d7-4bcf-b70f-a440436c210b",
      "organization_guid": "604bd59e-4139-4734-a3be-4e97836eb790",
      "host_ip": "10.0.16.15",
      "host_guid": "0455ec2b-11fa-41ab-9d1c-f3a575cd55ea"
    },
    "packet": {
      "direction": "ingress",
      "allowed": true,
      "src_ip": "10.255.73.3",
      "dst_ip": "10.255.73.2",
      "src_port": 46936,
      "dst_port": 8080,
      "protocol": "TCP",
      "mark": "0x1",
      "icmp_type": 0,
      "icmp_code": 0
    }
  }
}
```

### c2c denied

Kernel log:
```
Jul 24 19:21:51 localhost kernel: [1468798.671535] DENY_C2C_d5978989-1401-49ff-
IN=s-010255073003 OUT=s-010255073002
MAC=aa:aa:0a:ff:49:03:ee:ee:0a:ff:49:03:08:00 SRC=10.255.73.3 DST=10.255.73.2
LEN=60 TOS=0x00 PREC=0x00 TTL=63 ID=27922 DF PROTO=TCP SPT=37366 DPT=8081
WINDOW=27400 RES=0x00 SYN URGP=0 MARK=0x1
```

`iptables-logger` log:
```
{
  "timestamp": "1500924111.467581511",
  "source": "cfnetworking.iptables",
  "message": "cfnetworking.iptables.ingress-denied",
  "log_level": 1,
  "data": {
    "destination": {
      "container_id": "d5978989-1401-49ff-46cd-33e5",
      "app_guid": "bc6f229d-5e4a-4c41-a63f-e8795496c283",
      "space_guid": "b9f86312-a7d7-4bcf-b70f-a440436c210b",
      "organization_guid": "604bd59e-4139-4734-a3be-4e97836eb790",
      "host_ip": "10.0.16.15",
      "host_guid": "0455ec2b-11fa-41ab-9d1c-f3a575cd55ea"
    },
    "packet": {
      "direction": "ingress",
      "allowed": false,
      "src_ip": "10.255.73.3",
      "dst_ip": "10.255.73.2",
      "src_port": 37366,
      "dst_port": 8081,
      "protocol": "TCP",
      "mark": "0x1",
      "icmp_type": 0,
      "icmp_code": 0
    }
  }
}
```

