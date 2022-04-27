# Apps stop running after a deploy when using dynamic ASGs with icmp any rule

## Table of Contents
* [Context](#context)
* [Issue](#issue)
* [Expected Logs](#expected-logs)
* [Affected ASGs](#affected-asgs)
* [Affected Versions](#affected-versions)
* [Fixed Version](#fixed-version)
* [Check your ASGs](#check-your-asgs)
* [Temporary Mitigation](#temporary-mitigation)
* [Permanent Fix](#permanent-fix)

## Context
All ICMP packets have a type and a code, each represented by one byte. For ASGs
you can allow specific types and codes or you can allow all types and codes by
setting those properties to `-1` or `255`. When you allow all types and codes
this is called an “ICMP any rule”. See capi docs here for more information
about valid values for ASGs.

## Issue
When dynamic ASGs are enabled, the vxlan policy agent is unable to clean up
ICMP any rules. Undocumented iptables behavior with ICMP any rules causes a
cleanup failure, which causes a container creation failure, which prevents any
apps from starting. This results in a vxlan policy agent error `iptables: Bad
rule (does a matching rule exist in that chain?)` or `exit status 1: iptables:
No chain/target/match by that name.`.

## Expected Logs
If you are running into this issue, you should see one of the following logs in the vxlan-policy-agent.stdout.log.

If iptables logging is disabled:
```
{
  "timestamp": "2022-04-25T20:58:43.429519116Z",
  "level": "error",
  "source": "cfnetworking.vxlan-policy-agent",
  "message": "cfnetworking.vxlan-policy-agent.rules-enforcer.asg-5756ce1650920323410853.cleanup-rules",
  "data": {
    "error": "clean up parent chain: iptables call: running [/var/vcap/packages/iptables/sbin/iptables -t filter -D netout--c1c6107d-030b-47b7-6 -p icmp -m iprange --dst-range 0.0.0.0-255.255.255.255 -m icmp --icmp-type any -j ACCEPT --wait]: exit status 1: iptables: Bad rule (does a matching rule exist in that chain?).\n and unlock: <nil>",
    "session": "4.11"
  }
}
```

If iptables logging is enabled:
```
{
  "timestamp": "2022-04-25T20:00:10.555161129Z",
  "level": "error",
  "source": "cfnetworking.vxlan-policy-agent",
  "message": "cfnetworking.vxlan-policy-agent.rules-enforcer.asg-29730f1650916810536136.cleanup-rules",
  "data": {
    "error": "clean up parent chain: iptables call: running [/var/vcap/packages/iptables/sbin/iptables -t filter -D netout--7a79c4db-0126-4bee-4 -p icmp -m iprange --dst-range 0.0.0.0-255.255.255.255 -m icmp --icmp-type any -g netout--7a79c4db-0126-4--log --wait]: exit status 1: iptables: No chain/target/match by that name.\n and unlock: <nil>",
    "session": "4.30"
  }
}
```

## Affected ASGs
| protocol | type |
| --- | --- |
| icmp | -1 or 255 |


## Affected Versions
* silk-release 3.0.0 - 3.4.0

## Fixed Version
* silk-release 3.5.0

## Check your ASGs
Before upgrading to an affected version, check to see if you have ICMP any rules.

This script will detect ICMP any rules that will cause an error. This script
will print out the guid of the ASG that contains an ICMP any rule. If this
script detects ICMP any rules, then DO NOT upgrade to an affected version. If
this script does not detect ICMP any rules, then you are safe to upgrade.
```
per_page=1
next="/v3/security_groups?per_page=$per_page"

while [ "${next}" != "null" ]; do
    # return all guids that have ICMP type 255 or -1 (gets coerced to 'any' by iptables)
    guids=$(cf curl "${next}" | jq '.resources[] | select(.rules[] | select(.protocol == "icmp" and (.type == 255 or .type == -1))) | .guid' | uniq)
    for guid in $guids; do
        echo "$guid"
    done	

    fullUrl=$(cf curl "${next}" | jq '.pagination.next.href')
    # go to the next page (v3 returns full URL, so cut the protocol and host)
    next=$(echo "$fullUrl" | cut -d '/' -f 4-)
done
```


## Temporary Mitigation
Disable dynamic ASGs with this opsfile.

## Permanent Fix
Update to silk-release version 3.5.0.


