# CF Networking tolerance to etcd data loss

**Note:** This document only applies to cf-networking-deployments that use flannel.

CF Networking's batteries included CNI plugin uses flannel, which in turn
stores all the state for which cell has a given subnet in etcd.

The etcd-release README includes a section titled
[Failed Deploys, Upgrades, Split-Brain Scenarios, etc](https://github.com/cloudfoundry-incubator/etcd-release#failed-deploys-upgrades-split-brain-scenarios-etc).

This document describes how CF Networking is impacted when the etcd goes down
and the data directory is deleted as a recovery mechanism, or as a preventative
measure to avoid split-brains during deploys of Cloud Foundry in the first place.

We have explored scenarios where etcd data loss causes cells to lose network connectivity.

- Scenario #1: etcd data loss while flannel continues to run
- Scenario #2: flannel restarts on one cell after etcd data loss before lease is renewed
- Scenario #3: flannel restarts on one cell while etcd is down
- Scenario #4: new cell gets added after etcd comes back after data loss
- Scenario #5: subnet lease expires while etcd is down

In the five scenarios listed above, scenario 1 and 5 do not appear to be an issue.

Reducing flannel's subnet lease renewal interval to 1 minute significantly reduces the likelihood of
some of scenarios occurring.
[This is now the default for cf-networking-release](https://github.com/cloudfoundry-incubator/cf-networking-release/commit/e9a1b5facfc56c7413e5165b1c1639b1e9e8bf77).
Scenarios 2 and 4 are mitigated by reducing the subnet lease renewal time, though there
remains a 1 minute time window where if flannel does not renew its lease, loss
of network connectivity could still occur.

In particular, scenario 4 may be a concern when scaling up the number of Diego
cells in an environment. If etcd's data directory is wiped as part of the deployment
and new cells start coming online before existing cells can renew their lease, then
this would cause problems.

Scenario 3 remains an issue. We have a [story in our backlog](https://www.pivotaltracker.com/story/show/139995465) to address this.

Flannel-watchdog is a process that watches for these issues and starts failing
when it detects inconsistencies.  It also emits a metric called `flannel_watchdog.flanneldown`
which has a value of `1.0` when flannel-watchdog detects an issue.
If you get into a state where
flannel-watchdog is failing on a given cell, the cell must be recreated. See this [known
issue](known-issues.md#flannel-watchdog-failures).

## Scenarios


### Scenario #1: etcd data loss while flannel continues to run

1. etcd stops
2. etcd data directory is wiped out
3. etcd starts back up
4. flannel continues to run on all cells

Do existing cells keep connectivity?

Yes.


### Scenario #2: flannel restarts on one cell after etcd data loss before lease is renewed

1. etcd stops
2. etcd data directory is wiped out
3. etcd starts back up
4. flannel continues to run on all cells
5. ... flannel does not get a chance to renew lease (default renewal time is now every 1 minute)
6. flannel restarts on one cell

Do existing cells keep connectivity?

There is up to a 1 minute time window between etcd coming back up and flannel renewing it's lease.

If flannel restarts on a cell before it can renew it's lease, then connectivity between the cell 
where flanneld restarted and other cells is lost. Flannel acquires a new lease on a different subnet 
and then bridge does not match flannel device. This results in a "flannel-watchdog" failure on that cell.

Other cells remain able to connect to each other.


### Scenario #3: flannel restarts on one cell while etcd is down

1. etcd stops
2. etcd data directory is wiped out
3. flannel restarts on one cell
4. etcd starts back up
5. flannel restarts on cell again

Do existing cells keep connectivity?

Connectivity between the cell where flanneld restarted and other cells is lost. Flannel acquires a new lease on a different subnet 
and then bridge does not match flannel device. This results in a "flannel-watchdog" failure on that cell.

Other cells remain able to connect to each other.


### Scenario #4: new cell gets added after etcd comes back after data loss

1. etcd stops
2. etcd data directory is wiped out
3. etcd starts back up
4. flannel continues to run on all cells
5. new cell is added

Do existing cells keep connectivity?

If the new cell comes up before the existing cells can renew their subnet leases, then there is a chance
that the new cell will steal an existing cell's subnet lease.

In this case, the the existing cell will lose connectivity, and the new cell, will not be able to connect
to other cells until they do renew leases, which should happen within 1 minute.

### Scenario #5: subnet lease expires while etcd is down

1. etcd stops
2. etcd data directory is wiped out
3. subnet lease expires (default ttl is 24 hours)
3. etcd starts back up
4. flannel continues to run on all cells

Do existing cells keep connectivity?

Yes.
