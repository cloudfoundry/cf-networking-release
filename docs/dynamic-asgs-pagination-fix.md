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
