---
title: Known Issue - Policy Server ASG Syncer in Crash Loop
expires_at: never
tags: [cf-networking-release]
---

## Issue
The policy-server-asg-syncer fails when...
* running CF Networking Release version 3.68.0 or 3.69.0
* AND using MYSQL for the policy server DB
* AND dynamic ASGs are enabled

## Permanent Fix
A fix will be included in CF Networking Release 3.70.0. For deployments that are using MYSQL DBs with dynamic ASGS enabled, we suggest skipping CF Networking Release 3.68.0 and 3.69.0 and upgrading to CF Networking Release 3.70.0 or higher.

### Symptom
All instances of the policy-server-asg-syncer will be failing with the following log messages in
`/var/vcap/sys/log/policy-server-asg-syncer/policy-server-asg-syncer.stdout.log.`
```
{
  "timestamp": "2025-05-06T15:26:55.807028818Z",
  "level": "error",
  "source": "cfnetworking.policy-server-asg-syncer",
  "message": "cfnetworking.policy-server-asg-syncer.asg-sync-cycle",
  "data": {
    "error": "saving security group ########-####-####-####-############ (example_security_group): Error 1713 (HY000): Undo log record is too big."
  }
}
...
{
  "timestamp": "2025-05-06T15:26:55.811640274Z",
  "level": "error",
  "source": "cfnetworking.policy-server-asg-syncer",
  "message": "cfnetworking.policy-server-asg-syncer.exited-with-failure",
  "data": {
    "error": "Exit trace for group:\nasg-syncer exited with error: saving security group ########-####-####-####-############ (example_security_group): Error 1713 (HY000): Undo log record is too big.\nasg-lock exited with nil\n"
  }
}
```

## Root Cause
Migrations 82 and 83 both added functional indexes to the policy server database to make dynamic ASGs more performant.
However, when the size of “staging_spaces” or “running_spaces” is too large this can cause the undo log record to be too large, which will cause updates to the table to fail.


## Mitigation
You can manually drop the functional indexes that are causing issues.
1. Access the policy server db
```
DROP INDEX staging_spaces_idx ON security_groups;
DROP INDEX running_spaces_idx ON security_groups;
```
This is a safe procedure. The permanent fix has taken into account the fact that some DBs will be altered manually like this.
