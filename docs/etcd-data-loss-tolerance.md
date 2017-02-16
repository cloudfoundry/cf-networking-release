# CF Networking tolerance to etcd data loss

## Scenarios:

- Scenario #1: etcd data loss while flannel continues to run
- Scenario #2: flannel restarts on one cell after etcd data loss before lease is renewed
- Scenario #3: flannel restarts on one cell while etcd is down
- Scenario #4: new cell gets added after etcd comes back after data loss
- Scenario #5: new cell gets added while etcd is down
- Scenario #6: subnet lease expires while etcd is down

Details:

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
5. ... flannel does not get a chance to renew lease (default renewal time is every 23 hours)
6. flannel restarts on one cell

Do existing cells keep connectivity?

Connectivity between the cell where flanneld restarted and other cells is lost.
This results in a "flannel-watchdog" failure on that cell.
Flannel acquires a new lease on a different subnet and then bridge does not match flannel device.

Other cells remain able to connect to each other.


### Scenario #3: flannel restarts on one cell while etcd is down

1. etcd stops
2. etcd data directory is wiped out
3. flannel restarts on one cell
4. etcd starts back up
5. flannel restarts on cell again

Do existing cells keep connectivity?

At step 3, flannel will not start because it cannot reach etcd.
Then, after step 5, the behavior the same as scenario #2.


### Scenario #4: new cell gets added after etcd comes back after data loss

1. etcd stops
2. etcd data directory is wiped out
3. etcd starts back up
4. flannel continues to run on all cells
5. new cell is added

Do existing cells keep connectivity?

Yes, as long as the new cell does not "steal" an existing subnet lease.

Does the new cell have connectivity?

No, the sublease information for existing cells is not in etcd until they renew, which they only do every 23 hours by default.

So there are two issues:

  1. The new cell may take a sublease that conflicts with one of the existing cells.
  2. The new cell cannot reach the existing cells because it has no knowledge of what subnet they have.


### Scenario #5: new cell gets added while etcd is down

1. etcd stops
2. etcd data directory is wiped out
3. new cell is added (flannel fails to start on new cell)
4. etcd starts back up
5. flannel continues to run on all existing cells
6. flannel starts up on new cell

Do existing cells keep connectivity?

Yes, as long as the new cell does not "steal" an existing subnet lease.

Does the new cell have connectivity?

At step 3, flannel will not start because it cannot reach etcd.
Then, after step 6, the behavior the same as scenario #4.


### Scenario #6: subnet lease expires while etcd is down

1. etcd stops
2. etcd data directory is wiped out
3. subnet lease expires (default ttl is 24 hours)
3. etcd starts back up
4. flannel continues to run on all cells

Do existing cells keep connectivity?

Yes.

## Mitigations

Scenario #1: etcd data loss while flannel continues to run

OK: - This was already fine.

Scenario #2: flannel restarts on one cell after etcd data loss before lease is renewed

MIT1: Reduce lease renewal time from 23 hours to 1 minute
From our manual tests, during the 1 minute where etcd is being re-populated by cells renewing their leases,
it appears that connectivity among all cells remains OK.

Scenario #3: flannel restarts on one cell while etcd is down

MIT2 covers a spike that allows flannel to use it's local state in the subnet.env file to renew it's lease when it starts up.
MIT1 alone does not solve this, because flannel "forgets" it's lease when it restarts,
and then it will not find it in etcd after etcd comes back up.

Scenario #4: new cell gets added after etcd comes back after data loss

MIT1: Reduce lease renewal time from 23 hours to 1 minute. As long as new cell doesn't come up existing cells renew their leases,
this should be fine. That is much less likely with a 1 minute lease renewal time.

Scenario #5: new cell gets added while etcd is down

MIT1: This is the same as scenario 4.

Scenario #6: subnet lease expires while etcd is down

OK: This was already fine.

### MIT1: Reduce lease renewal time (by increasing --subnet-lease-renew-margin`)

This is now done as of [this commit](https://github.com/cloudfoundry-incubator/cf-networking-release/commit/e9a1b5facfc56c7413e5165b1c1639b1e9e8bf77).

An existing config option exists that can allow us to have flannel renew it's lease more frequently.
The lease expires after 24 hours.  This is hard-coded in flannel.  But the renew margin controls when the next renewal occurs
(backdated from the expiration date). Setting `--subnet-lease-renew-margin` to `1439` (24 hours * 60 minutes - 1) effectively
makes flannel renew it's lease every minute.

### MIT2: Use subnet.env to restore state in etcd

Make PR to flannel, or add something to our own start script that puts the information from subnet.env back in etcd
before flannel starts up again.

See spikes on [wip-read-from-subnet-file](https://github.com/coreos/flannel/compare/master...cf-container-networking:wip-read-from-subnet-file).
