---
title: Dynamic ASGs
expires_at: never
tags: [cf-networking-release]
---

<!-- vim-markdown-toc GFM -->

* [Dynamic ASGs](#dynamic-asgs)
  * [Overview](#overview)
    * [Affected Versions](#affected-versions)
    * [Without Dynamic ASGs](#without-dynamic-asgs)
    * [With Dynamic ASGs](#with-dynamic-asgs)
  * [FAQ](#faq)
    * [1. I'm not using silk. How does this change things?](#1-im-not-using-silk-how-does-this-change-things)
    * [2. How do I enable dynamic ASGs?](#2-how-do-i-enable-dynamic-asgs)
    * [3. How do I disable dynamic ASGs?](#3-how-do-i-disable-dynamic-asgs)
    * [4. I don't like waiting a few minutes for rules to be enforced. Can it be faster?](#4-i-dont-like-waiting-a-few-minutes-for-rules-to-be-enforced-can-it-be-faster)
    * [5. Did we stop passing ASG information through the LRP?](#5-did-we-stop-passing-asg-information-through-the-lrp)
    * [6. If I create an ASG and immediately push an app, will the ASGs be up-to-date?](#6-if-i-create-an-asg-and-immediately-push-an-app-will-the-asgs-be-up-to-date)
    * [7. What about windows?](#7-what-about-windows)
    * [8. Why are my security-group updates taking a very long time to be detected and synced through to policy-server?](#8-why-are-my-security-group-updates-taking-a-very-long-time-to-be-detected-and-synced-through-to-policy-server)
    * [Problem:](#problem)
    * [Solution Illustration (bolded pages are the ones we actually query capi for):](#solution-illustration-bolded-pages-are-the-ones-we-actually-query-capi-for)
    * [Q&A:](#qa)

<!-- vim-markdown-toc -->
# Dynamic ASGs 

## Overview

ASG - Application Security Group. These network egress rules are an allow list
of IPs and protocols that apps are allowed to access.

### Affected Versions
As of cf-networking-release 3.0.0 and silk-release 3.0.0 dynamic ASG enforcement
is available and on by default.

### Without Dynamic ASGs
Before we talk about dynamic ASGs, it is important to understand what existed first.

When Dynamic ASGs enforcement is _not_ enabled then ASG rule enforcement
requires app restarts in order for rules to take effect. Cloud Foundry operators
with large installations and distributed teams spend an enormous amount of time
trying to wrangle developer teams to restart their apps when global rules
change. This experience is also at odds with expectations, as developers expect
these changes to take effect immediately with no extra action on their part.

![non-dynamic ASG architecture
diagram](asg-enforcement-during-container-create-architecture.png)
<!-- Use exiftool and check Url metadata for PNG to see where this PNG came from --!>

**Steps taken to create and enforce ASGs**
1. An operator creates and binds and ASG.
1. An app dev restarts their app.
1. The cloud controller creates a desired LRP with the ASG rules.
1. The rep gets desired LRP information when launching containers.
1. The executor calls out to garden to start a container.
1. Garden calls out to the silk CNI plugin to set up the networking for the
   container.
1. The silk CNI plugin creates ASGs as iptables netout rules.

### With Dynamic ASGs
When Dynamic ASGs enforcement _is_ enabled then ASG rule enforcement _does not_
require app restarts in order for rules to take effect. Within a few minutes
rules will be automatically enforced.

![dynamic ASG architecture
diagram](dynamic-asg-enforcement-architecture.png)
<!-- Use exiftool and check Url metadata for PNG to see where this PNG came from --!>

**Steps taken to create and enforce dynamic ASGs**
1. An operator creates and binds and ASG.
1. Cloud Controller updates its internal reference to the last time ASGs were modified. **NOTE** This only occurs on `/v3` API endpoints.
1. A new job, the Policy Server ASG Syncer, polls Cloud Controller for the last time ASGs were updated. If changes have been made, it syncs all ASGs.
   The syncer saves all the ASGs in the policy server DB.  This poll interval is
   controlled by the syncer's bosh property
   [`asg_poll_interval_seconds`](https://github.com/cloudfoundry/cf-networking-release/blob/0c5029c88e9f61bb94829b4e0b8ed6732f30f9f0/jobs/policy-server-asg-syncer/spec#L31-L33).
1. The vxlan policy agent (VPA) polls the policy server for c2c policies _and_
   ASG rules relavent for the apps on its cell. This poll interval is controlled
   by the VPA's bosh property [`asg_poll_interval_seconds`](https://github.com/cloudfoundry/silk-release/blob/f1606499925ff94bc036a641d688967c7af1fef4/jobs/vxlan-policy-agent/spec#L59-L61).
1. If the ASGs have changed, then the VPA creates the new ASGs as iptables
   netout rules.

## FAQ

### 1. I'm not using silk. How does this change things?
This implementation of dynamic ASGs is only for silk users.

### 2. How do I enable dynamic ASGs?
To enable dynamic ASGs the following two properties must be set:
| job name | property name | value to enable | link to code |
| --- | --- | --- | --- |
| policy-server-asg-syncer | disable | false | [link](https://github.com/cloudfoundry/cf-networking-release/blob/0c5029c88e9f61bb94829b4e0b8ed6732f30f9f0/jobs/policy-server-asg-syncer/spec#L27-L29) |
| vxlan-policy-agent | enable_asg_syncing | true | [link](https://github.com/cloudfoundry/silk-release/blob/f1606499925ff94bc036a641d688967c7af1fef4/jobs/vxlan-policy-agent/spec#L55-L57) |

If you are using cf-deployment then dynamic ASGs are on by default.

### 3. How do I disable dynamic ASGs?
You can disable dynamic ASGs by using [this opsfile](https://github.com/cloudfoundry/cf-deployment/blob/604b5822259d8c889c3cdc4a2723af0e636570bb/operations/disable-dynamic-asgs.yml).

### 4. I don't like waiting a few minutes for rules to be enforced. Can it be faster?
If you want to speed up the dynamic enforcement you can update the following
bosh properties. Lowering these values will have performance implications.

| job name | property name | default | link to code|
| --- | --- | --- | --- |
| policy-server-asg-syncer | asg_poll_interval_seconds | 60 | [link](https://github.com/cloudfoundry/cf-networking-release/blob/0c5029c88e9f61bb94829b4e0b8ed6732f30f9f0/jobs/policy-server-asg-syncer/spec#L31-L33) |
| vxlan-policy-agent | asg_poll_interval_seconds | 60 | [link](https://github.com/cloudfoundry/silk-release/blob/f1606499925ff94bc036a641d688967c7af1fef4/jobs/vxlan-policy-agent/spec#L59-L61) |

### 5. Did we stop passing ASG information through the LRP?
No. ASGs are still passed through Cloud Controller to BBS to Rep to Garden to
Silk CNI. This allows other CNIs to keep exisiting behavior. However, Silk CNI no longer uses this data to 
enforce ASGs when dyanmic ASGs are enabled.

### 6. If I create an ASG and immediately push an app, will the ASGs be up-to-date?
The ASGs are synced to the policy server DB within 60 seconds (by default) of creating or updating an ASG.
When the app container is created, the silk CNI calls the force-asg-sync endpoint on the vxlan-policy-agent.
This way the vxlan-policy-agent gets the most up-to-date ASG information from the policy server when a container is created.

There is a chance of hitting a race condition if you create an ASG and immediately push an app.
By default this window is at most 60 seconds. If you run into this issue, try pushing your app again.
By the time you push a second time the ASGs should be synced.

We believe that this race condition is acceptable because we believe that users do not update their ASGs very often. If this assumption is incorrect and you are running into this issue, please open an issue and let us know.

### 7. What about windows?
Dynamic ASGs is a linux-only feature. Currently there is no vxlan-policy-agent
equivalent for windows cells. If this is a feature you are interested in, please
open an issue as a feature request.

### 8. Why are my security-group updates taking a very long time to be detected and synced through to policy-server?

Because only `/v3/security_groups` endpoints update Cloud Controller's internal timestamp for when ASG info has changed, any
ASG create/update/delete activity initiated using the `/v2/security_groups` will go unnoticed by `policy-server-asg-syncer`.
However, upon the next ASG modification using the CF API v3 endpoints, a full sync will occur and the changes made using the
CF API v2 endpoints will be pulled in at this time. As the CF API v2 has been deprecated for some
time, there are no plans to address this, and it is advised to update any integrations to use the CF API v3 endpoints
instead.

Purpose of this document is to explain the algorithm in policy-server's CCClient which polls capi for security groups.

Future versions of cf-networking will migrate the source of truth for security groups to policy-server and elimintate the need to poll capi for ASGs (after which this document can be deleted).

### Problem:
Guard against the following scenario:

* CAPI has ASGs called: "a", "b", "c", "d", "e", "f"
* ASG poller gets the first page of ASGs: "a", "b", "c"
* User deletes ASG "a"
* ASG poller gets the second page of ASGs: "e", "f"
* ASG poller misses ASG "d"
* ASG "d" is now not enforced for an hour until the next poll cycle.

This could break apps and break pushing new apps
Given that we support multi-tenant systems, a malicious actor could do this to break other people's apps

Solution introduced in `policy-server's` [cc_client.GetSecurityGroups() method](https://github.com/cloudfoundry/cf-networking-release/blob/develop/src/code.cloudfoundry.org/policy-server/cc_client/client.go#L372).


###  Solution Illustration (bolded pages are the ones we actually query capi for):
First Query (page=1, page_size=5000):<br>
**page1: 0-4999**, page2: 5000-9999, page3: 10000-14999, page4: 15000-19999<br><br>
Second Query (page=2, page_size=4999):<br>
page1: 0-4998, **page2: 4999-9997**, page3: 9998-14996, page4: 14997 - 19995, page5: 19996-19999<br><br>
Third Query (page=3, page_size=4998):<br>
page1: 0-4997, page2: 4998-9995, **page3: 9996-14993**, page4: 14994 - 19991, page5: 19992 - 19999<br><br>
Fourth Query ([age=4, page_size=4997):<br>
page1: 0-4996, page2: 4997-9993,  page3: 9994 - 14990, **page4: 14991 - 19987**, page5: 19988 - 19999<br><br>
Fifth Query (page=5, page_size=4996):<br>
page1: 0-4995, page2: 4996 - 9991, page3: 9992 - 14987, page4: 14988 - 19983, **page5: 19984 - 19999**<br><br>

On the second query, we check that index0 of the second query (4999) was the same as the last index of the first query (4999).<br>
On the third query we check that index1 of the third query (9997) was the same as the last index of the second query (9997).<br>
On the fourth query, we check that index2 of the fourth query (14993) was the same as the last index of the third query (14993).<br>
On the fifth query, we check that index3 of the fifth query (19987) was the same as the last index of the fourth query (19987).<br>


###  Q&A:

Q: Why is this complex pagination necessary?<br>
A: We need to detect any changes (deletions) in the capi response that happened after the start of the poll cycle. We sort by `created_by`, so any additions are at the end. However deletions are likely somewhere in the middle and cause all ASGs following them to be shifted up, causing us to miss non-deleted ASGs (see "Guard against the following scenario, above).

Q: Why do we have to decrement page size for each page?<br>
A: We want the create an overlap between the response of the last query with the present query.<br>

Capi lets us set `page`, `per_page`, and `order_by` query parameters. Given the query parameters we have access to, decrementing `page_size` as we increment page is is the only way we can create an overlap (see Solution Illustration, above).


Q: Why do we have to increment the index of the ASG we are inspecting?<br>
A: As we decrement, the overlap we create will get bigger and bigger. We don't need to compare all the contents of the overlap, just the last overlapping ASG (see Solution Illustration, above).

Q: What happens when there are +5000 pages?<br>
A: Actualy, once there are *2500* pages, we run out of space in the result set. (This is because page size goes DOWN at the same time that index goes UP). This would be a problem; however even our customers with the largest numbers of ASGs don't have 2500 pages of 5000+4999+4998...	ASGs per page.

Q: How long will this algorithm be in place?<br>
A: Our first priority for TAS 2.14 is to refactor ASGs so that policy-server, not capi, is the source of truth for ASGs. This will remove this feature's dependcy on capi and we will be able to remove this algorithm.

