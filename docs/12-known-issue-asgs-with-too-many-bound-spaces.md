---
title: Know Issue - Policy Server Mysql DB Failures when a Non-Global ASG is Bound to More than 148 Spaces
expires_at: never
tags: [cf-networking-release]
---


## Issue
The policy-server pre-start fails when upgrading to cf-networking-release version 3.68.0 or higher when using mysql.

### Symptom 1: failing on migration 82

```
{
  "timestamp": "2025-05-01T10:45:10.131469969Z",
  "level": "error",
  "source": "cfnetworking.policy-server-migrate-db",
  "message": "cfnetworking.policy-server-migrate-db.failed migrating and populating tags, retrying",
  "data": {
    "error": "perform migrations: executing migration: executor.Exec: Error 3906 (HY000): Exceeded max total length of values per record for multi-valued index staging_spaces_idx by 84 bytes. handling 82"
  }
}
```

### Symptom 2: failing on migration 83

```
{
  "timestamp": "2025-05-01T10:45:10.131469969Z",
  "level": "error",
  "source": "cfnetworking.policy-server-migrate-db",
  "message": "cfnetworking.policy-server-migrate-db.failed migrating and populating tags, retrying",
  "data": {
    "error": "perform migrations: executing migration: executor.Exec: Error 3906 (HY000): Exceeded max total length of values per record for multi-valued index running_spaces_idx by 84 bytes. handling 83"
  }
}
```

## If you have not upgraded to an impacted version yet, test to see if you will be impacted

### Option 1: Use the CLI and API

```
security_groups="$(cf curl /v3/security_groups)"
pages="$(echo ${security_groups} | jq .pagination.total_pages)"

for (( p=1; p<=${pages}; p++ ))
do
    security_groups="$(cf curl /v3/security_groups?page=${p})"
    echo "${security_groups}" | jq '[.resources[] | select(.relationships.staging_spaces.data | length >= 148)] | map({guid, name, staging_spaces_count: (.relationships.staging_spaces.data | length)})'
    echo "${security_groups}" | jq '[.resources[] | select(.relationships.running_spaces.data | length >= 148)] | map({guid, name, running_spaces_count: (.relationships.running_spaces.data | length)})'
done
```

If any results are returned, then you will run into this bug and you should follow the mitigations. Below is an example of what results would look like from the script above.
```
[
  {
    "guid": "14ad7fc8-27c2-4456-9641-3d9f8cffb1c1",
    "name": "too_many_staging_spaces_example",
    "staging_spaces_count": 160
  }
]
[
  {
    "guid": "14ad7fc8-27c2-4456-9641-3d9f8cffb1c1",
    "name": "too_many_running_spaces_example",
    "running_spaces_count": 170
  }
]
```


### Option 2: Query the database
1. Connect to the policy server db. 
1. Run the following queries.
```
# for mysql
select name from security_groups WHERE JSON_LENGTH(staging_spaces) > 148;
select name from security_groups WHERE JSON_LENGTH(running_spaces) > 148;

# for postgres
select name from security_groups WHERE json_array_length(staging_spaces::json) > 148;
select name from security_groups WHERE json_array_length(running_spaces::json) > 148;
```

If either of those queries return any rows, then you will run into this bug and you should follow the mitigations.

## Root Cause
Migrations 82 and 83 both add functional indexes to the policy server database to make dynamic ASGs more performant.
However, when the size of “staging_spaces” or “running_spaces” is too large the functional index will fail to be created,
and thus the migration will fail. This causes the pre-start script to fail. 

The “staging_spaces” and “running_spaces” columns become too large when a single ASG is bound to more than 148 individual spaces for that lifecycle. 

## Resolution
There is no permanent fix at this time (05/06/2025).

## Mitigations
### Mitigation Option 1: Use global ASGs
Global ASGs have an empty array for "running_spaces" and "staging_spaces" in the database so they will never trigger this issue.
1. Make and bind a new global ASG with all the same rules as the problematic ASG.
2. Delete the problematic ASG

### Mitigation Option 2: Make multiple ASGs with the same rules
Instead of binding one ASG to 148+ spaces, make 2 identical ASGs and bind them to <149 spaces each.

### Check Mitigation
If you have already deployed, or attempted to deploy, cf-networking-release version 3.68.0 or higher you can run the policy server migrations manually.

```
# commands run on diego_database bootstrap VM

# become root
sudo su - 

# make sure you are on the bootstrap VM, if this file is empty then you are on the wrong VM
cat /var/vcap/jobs/policy-server/bin/pre-start

# run the pre-start script. It will log output and will migrate the db
/var/vcap/jobs/policy-server/bin/pre-start
```

### Breaking Glass Mitigation - in dire cases only
If the you have to continue an upgrade urgently and can't do either of the mitigations listed, you can force skip these migrations.
1. Access the policy server db
2. Add these rows manually so it will fake as if migrations 82 and 83 have run.
```
insert into gorp_migrations (id, applied_at) values (82, NOW());
insert into gorp_migrations (id, applied_at) values (83, NOW());
```

⚠️ WARNING: If you skip migrations now, you should still work towards running the mitigations listed above.
Once you have done the mitigations you can delete these rows from the migration database and follow the steps above to run the migrations manually. 
 

