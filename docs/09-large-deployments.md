---
title: Large Deployment Best Practices
expires_at: never
tags: [cf-networking-release,silk-release]
---

<!-- vim-markdown-toc GFM -->

* [Large Deployment best practices for CF-Networking and Silk Release](#large-deployment-best-practices-for-cf-networking-and-silk-release)
  * [Problem 0: Default overlay IP CIDR block too small when there are 250+ diego cells](#problem-0-default-overlay-ip-cidr-block-too-small-when-there-are-250-diego-cells)
    * [Symptoms](#symptoms)
    * [Solution](#solution)
  * [Problem 1: Silk Daemon uses too much CPU](#problem-1-silk-daemon-uses-too-much-cpu)
    * [Symptoms](#symptoms-1)
    * [Reason](#reason)
    * [Solution](#solution-1)
  * [Problem 2: ARP Cache on diego-cell not large enough](#problem-2-arp-cache-on-diego-cell-not-large-enough)
    * [Symptoms](#symptoms-2)
    * [Reason](#reason-1)
    * [Solution](#solution-2)
  * [Problem 3: Too frequent and in-sync polling from the silk-daemon and the vxlan-policy-agent](#problem-3-too-frequent-and-in-sync-polling-from-the-silk-daemon-and-the-vxlan-policy-agent)
    * [Symptoms](#symptoms-3)
    * [Reason](#reason-2)
    * [Solution](#solution-3)
  * [Problem 4: Reaching the Upper Limit of Network Policies](#problem-4-reaching-the-upper-limit-of-network-policies)
    * [Summary](#summary)
    * [Reason](#reason-3)
    * [Scenario 1 - policies with no overlapping apps](#scenario-1---policies-with-no-overlapping-apps)
    * [Scenario 2 - policies with overlapping apps](#scenario-2---policies-with-overlapping-apps)
  * [Problem 5: NAT Gateway port exhaustion](#problem-5-nat-gateway-port-exhaustion)
    * [Symptoms](#symptoms-4)
    * [Reason](#reason-4)
    * [Solution](#solution-4)

<!-- vim-markdown-toc -->
# Large Deployment best practices for CF-Networking and Silk Release

Some users have larger deployments than we regularly test with. We have heard of
large deployments with 500-1000 diego cells.  These deployments have specific
considerations that smaller deployments don't need to worry about.

Please submit a PR or create an issue if you have come across other large
deployment considerations.

## Problem 0: Default overlay IP CIDR block too small when there are 250+ diego cells

### Symptoms

The silk daemon on some diego cells fails because it cannot get a lease.

### Solution

Increase the size of the `silk-controller.network` CIDR in the [silk controller
spec](https://github.com/cloudfoundry/silk-release/blob/develop/jobs/silk-controller/spec).

## Problem 1: Silk Daemon uses too much CPU
### Symptoms

The silk daemon begins using too much CPU on the cells. This causes the app
health checks to fail, which causes the apps to evacuate the cell.

### Reason

The silk daemon is deployed on every cell. It is in charge of getting the IP
leases for every other cell from the silk controller. The silk daemon calls out
to the silk controller every 5 seconds (by default) to get updated lease
information. Every time it gets new information the silk daemon does some linux
system calls to set up the networking. This can take a long time (relatively)
and get expensive when there are a lot of cells with new leases. This causes the
silk daemons to use a lot of CPU.

### Solution

Change the property `lease_poll_interval_seconds` on the silk-daemon job to be
greater than 5 seconds. This will cause the silk-daemon to poll the
silk-controller less frequently and thus make linux system calls less
frequently. However, increasing this property means that when a cell gets a new
lease (this happens when a cell is rolled, recreated, or for whatever reason it
doesn't renew it's lease properly) it will take longer for the other cells to
know how to route container-to-container traffic to it. To start with, we
suggest setting this property to 300 seconds (5 minutes). Then you can tweak
accordingly.

## Problem 2: ARP Cache on diego-cell not large enough
[Github issue](https://github.com/cloudfoundry/cf-networking-release/issues/54)

### Symptoms

Silk daemon fails to converge leases. Errors in the silk-daemon logs might look
like this:

```json
{
   "timestamp": "TIME",
   "source": "cfnetworking.silk-daemon",
   "message": "cfnetworking.silk-daemon.poll-cycle",
   "log_level": 2,
   "data": {
      "error":"converge leases: del neigh with ip/hwaddr 10.255.21.2 : no such file or directory"
   }
}
```

Also kernel logs might look like this:

```
neighbour: arp_cache: neighbor table overflow
```

### Reason

ARP cache on the diego cell is not large enough to handle the number of entries
the silk-daemon is trying to write.

### Solution

Increase the ARP cache size on the diego cells.

1. Look at the current size of your ARP cache
    - ssh onto a diego-cell and become root
    - inspect following kernel variables
    ```bash
    sysctl net.ipv4.neigh.default.gc_thresh1
    sysctl net.ipv4.neigh.default.gc_thresh2
    sysctl net.ipv4.neigh.default.gc_thresh3
    ```

1. Manually increase ARP cache size on the cell. This is good for fixing the
   issue in the moment, but isn't a good long term soluation because the values
   will be reset when the cell is recreated.
   - set new, larger values for the kernel variables. These sizes were used successfully for a deployment of ~800 cells.
     ```bash
     sudo sysctl -w net.ipv4.neigh.default.gc_thresh3=8192;
     sudo sysctl -w net.ipv4.neigh.default.gc_thresh2=4096;
     sudo sysctl -w net.ipv4.neigh.default.gc_thresh1=2048;
     ```

1. For a more permanent solution, set these variables by adding the
   [os-conf-release](https://github.com/cloudfoundry/os-conf-release) sysctl job
   to the deigo-cell instance group. A conf file will be autogenerated into
   `/etc/stsctl.d/71-bosh-os-conf-sysctl.conf`.
   - the manifest changes will look similar to this:
     ```yaml
     instance_groups:
     - name: diego-cell
       jobs:
       - name: sysctl
         properties:
            sysctl:
            - net.ipv4.neigh.default.gc_thresh3=8192
            - net.ipv4.neigh.default.gc_thresh2=4096
            - net.ipv4.neigh.default.gc_thresh1=2048
         release: os-conf

     ...

     releases:
     - name: "os-conf"
       version: "20.0.0"
       url: "https://bosh.io/d/github.com/cloudfoundry/os-conf-release?v=20.0.0"
       sha1: "a60187f038d45e2886db9df82b72a9ab5fdcc49d"
     ```
 
## Problem 3: Too frequent and in-sync polling from the silk-daemon and the vxlan-policy-agent
### Symptoms
* All silk-daemons can't connect to the silk-controller
* Silk-controller is overwhelmed with connections
* All vxlan-policy-agents can't connect to the network-policy-server
* Network-policy-server is overwhlemed with connections

### Reason
The silk-daemon and the vxlan-policy-agent both live on the Diego Cell. The silk-daemon polls the silk-controller [every 30 seconds by default](https://github.com/cloudfoundry/silk-release/blob/develop/jobs/silk-daemon/spec#L42-L44). The vxlan-policy-agent polls the network-policy-server [every 5 seconds by default](https://github.com/cloudfoundry/silk-release/blob/develop/jobs/vxlan-policy-agent/spec#L42-L44). If there is a high "max_in_flight" set for the Diego Cell instance group, then it is possible for many cells (50+) to start at the same time. This means that many silk-daemons and vxlan-policy-agents start polling at nearly the exact same time. This can overwhelm the jobs that they are polling.

### Solution
* Lower max in flight
* Increase polling interval for the [silk-daemon](https://github.com/cloudfoundry/silk-release/blob/develop/jobs/silk-daemon/spec#L42-L44) and/or the [vxlan-polixy-agent](https://github.com/cloudfoundry/silk-release/blob/develop/jobs/vxlan-policy-agent/spec#L42-L44)

## Problem 4: Reaching the Upper Limit of Network Policies

To our knowledge no one has actually run into this problem, even in the largest of deployments. However our team is often asked about this, so it seems important to cover it.

### Summary 

The quick answer is that you are limited to 65,635 _apps_ used in network policies. This results in _at least_ 32,767 network policies. 

### Reason

Container networking policies are implemented using [linux marks](https://www.linuxtopia.org/Linux_Firewall_iptables/x4368.html). Each source and destination app in a networking policy is assigned a mark at the policy creation time. If the source or destination app already has a mark assigned to it from a different policy, then the app uses that mark and does not get a new one. The overlay network for container networking uses VXLAN. VXLAN limits the marks to 16-bits. With 16 bits there are 2^16 (or 65,536) distinct values for marks. The first mark is saved and not given to apps, so that results in 65,535 marks available for apps.

### Scenario 1 - policies with no overlapping apps
Let's imagine that there are 65,535 _different_ apps. A user could create 32,767 network policies from appA --> appB, where appA and appB are only ever used in ONE network policy. Each of the 32,767 policies includes two apps (the source and the destination) and each of those apps needs a mark. This would result in 65,634 marks. This would reach the upper limits of network policies. 

### Scenario 2 - policies with overlapping apps
Let's imagine that there are 5 apps. Let's say a user wants all 5 apps to be able to talk to everyother app. This would result in 25 network policies. However, this would only use up 5 marks (one per app). There are still 65,630 marks available for other apps. This scenario shows how the more "overlapping" the policies are, the more policies you can have.

## Problem 5: NAT Gateway port exhaustion

### Symptoms

* Apps timeout while trying to connect to particular endpoints
* Multiple port allocation issues on NAT Gateways
* Multiple apps try to open multiple connections to a single service

### Reason

Each foundation has a finite number of NAT Gateways each of which can open up to 2<sup>16</sup> = 65536 ports per destination IP and destination port ([explanation](https://stackoverflow.com/questions/2332741/what-is-the-theoretical-maximum-number-of-open-tcp-connections-that-a-modern-lin)). By default the number of outbound connections per app are not limited. This is grounds for the noisy neighbour problem where bad apps exhaust the number of connections that could be opened to a given service thus blocking access to it. The issue could be fixed by applying hard limits on the number of connections that could be opened by each app in order to incapacitate the badly behaving ones.

NAT Gateway ports could be exhausted in another way which is easier to implement. Instead of opening long lived connections a bad app could frequently open short lived ones. Because of the way TCP works, after each connection is closed the client-side ports would be kept in a TIME_WAIT state for a few minutes before they are released ([explanation](https://superuser.com/questions/173535/what-are-close-wait-and-time-wait-states)). The way to fix this is by applying rate limits on the number of outbound connections.

### Solution

Currently the implementation of hard limits is blocked by a [netfilter issue](https://unix.stackexchange.com/questions/654525/how-can-i-prevent-iptables-connlimit-counter-from-resetting-each-time-an-iptable).

Rate limiting on the other hand is implemented as part of the `silk-cni` job and could be used through optional parameters under the `outbound_connections` field:

- `limit` is an on/off switch for the feature.
- `burst` is the maximum number of outbound connections per destination host allowed to be opened at once per container.
- `rate_per_sec` is the maximum number of outbound connections to be opened per second per destination host per container given that the burst is exhausted.

Additionally `iptables` logging of connections denied due to rate limits is available when `iptables_logging` is set to `true`. Such a log message is expected to have a prefix in the format `DENY_ORL_<container-id>`.
