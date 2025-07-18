---
title: Network Policy Database
expires_at: never
tags: [cf-networking-release]
---

<!-- vim-markdown-toc GFM -->

* [Network Policy Database](#network-policy-database)
   * [<a name="access-db"></a> How to access an internal database](#a-nameaccess-dba-how-to-access-an-internal-database)
   * [<a name="table-overview"></a> Table Overview](#a-nametable-overviewa-table-overview)
   * [<a name="network-policy-tables"></a> Network Policy Related Tables](#a-namenetwork-policy-tablesa-network-policy-related-tables)
      * [<a name="groups-table"></a> Groups](#a-namegroups-tablea-groups)
      * [<a name="destinations-table"></a> Destinations](#a-namedestinations-tablea-destinations)
      * [<a name="policies-table"></a> Policies](#a-namepolicies-tablea-policies)
   * [<a name="network-policy-example"></a> Networking Policy Example](#a-namenetwork-policy-examplea-networking-policy-example)
   * [<a name="migrations-tables"></a> Migration Related Tables](#a-namemigrations-tablesa-migration-related-tables)
      * [<a name="gorp-mirations-table"></a> gorp_migrations](#a-namegorp-mirations-tablea-gorp_migrations)
      * [<a name="gorp-lock-table"></a> gorp_lock](#a-namegorp-lock-tablea-gorp_lock)

<!-- vim-markdown-toc -->

# Network Policy Database

This document is intended to help people who are poking around the `network_policy` database. There are a lot of tables (most of them empty) and the names of the rows are not the most intuative. This doc should help you connect to the database and understand what you find there.

## <a name="access-db"></a> How to access an internal database
1. Bosh ssh onto the VM where the `policy-server` is running. You can figure out what machine by running `bosh is --ps | grep policy-server`.
2. Grab the mysql config. 
   ```
   $ cat /var/vcap/jobs/policy-server/config/policy-server.json | grep \"database\" -A 11
   
    "database": {
       "type": "mysql",
       "user": "USER",
       "password": "PASSWORD",
       "host": "HOST",
       "port": PORT,
       "timeout": 120,
       "database_name": "network_policy",
       "require_ssl": true,
       "ca_cert": "/var/vcap/jobs/policy-server/config/certs/database_ca.crt",
       "skip_hostname_validation": false
    },
   ```
   
3. Bosh ssh onto the database VM.
4. Connect to the mysql instance.
   ```
   /var/vcap/packages/pxc/bin/mysql -u USER -p -h HOST -D network_policy
   ```

## <a name="table-overview"></a> Table Overview

Below are all of the tables in the `network_policy` database.

| Table Name | Description  |
|---|---|
| destinations | List of metadata about network policies. |
| gorp_lock  | Locking mechanism for running migrations. |
| gorp_migrations  | Record of which migrations have been run. |
| groups  | List of all apps that are either the source or destination of a network policy. |
| policies  | List of source apps and destination metadata for network policies. |
| policies_info | A single row indicating the last update time of any network policy, used to save on DB queries from vxlan-policy-agent |
| security_groups | Lists all security groups defined in CAPI. This is populated by policy-server-asg-syncer, and is not a source of truth. |
| security_groups_info | A single row indicating the last update time of any security group info, used to save on DB queries from vxlan-policy-agent |
| running_security_groups_spaces | A join table associating security groups to the spaces they are bound to for running lifecycle workloads. |
| staging_security_groups_spaces | A join table associating security groups to the spaes they are bound to for staging lifecycle workloads. |

The following tables were related to dynamic egress, which has been removed
from the codebase. These tables should no longer present in your database as of
v3.6.0.

| Table Name | Description  |
|---|---|
| apps | Related to dyanmic egress which is deprecated. Should be empty. |
| defaults |  Related to dyanmic egress which is deprecated. Should be empty. |
| destination_metadatas  |  Related to dyanmic egress which is deprecated. Should be empty.  |
| egress_policies  |  Related to dyanmic egress which is deprecated. Should be empty. |
| groups  | List of all apps that are either the source or destination of a network policy. |
| ip_ranges  | Related to dyanmic egress which is deprecated. Should be empty. |
| spaces   | Related to dyanmic egress which is deprecated. Should be empty. |
| terminals  | Related to dyanmic egress which is deprecated. Should be empty.  |

## <a name="network-policy-tables"></a> Network Policy Related Tables

There are three tables related to cf networking policies: policies, groups, and destinations. 

### <a name="groups-table"></a> Groups

There is an entry in the group table for each app involved in network policies. A group is created for both the source and destination app for a policy.

This table is auto-populated with 65,535 rows with the value `NULL` in the `guid` column. This is the limit of how many apps may be involved with network policies.

```
mysql> describe groups;
+-------+--------------+------+-----+---------+----------------+
| Field | Type         | Null | Key | Default | Extra          |
+-------+--------------+------+-----+---------+----------------+
| id    | int(11)      | NO   | PRI | NULL    | auto_increment |
| guid  | varchar(255) | YES  | UNI | NULL    |                |
| type  | varchar(255) | YES  | MUL | app     |                |
+-------+--------------+------+-----+---------+----------------+
```
| Field | Note  |
|---|---|
| id | "id" is the primary key for this table. |
| guid | "guid" is the app guid.  |
| type | "type" was added to diferentiate between policies for orgs, spaces, and apps. However we never implemented network policies for orgs or spaces. This value is always app. |


### <a name="destinations-table"></a> Destinations
There is an entry in the destinations table for every network policy. This describes exactly what access is allowed to the destination app.

```
mysql> describe destinations;
+------------+--------------+------+-----+---------+----------------+
| Field      | Type         | Null | Key | Default | Extra          |
+------------+--------------+------+-----+---------+----------------+
| id         | int(11)      | NO   | PRI | NULL    | auto_increment |
| group_id   | int(11)      | YES  | MUL | NULL    |                |
| port       | int(11)      | YES  |     | NULL    |                |
| protocol   | varchar(255) | YES  |     | NULL    |                |
| start_port | int(11)      | YES  |     | NULL    |                |
| end_port   | int(11)      | YES  |     | NULL    |                |
+------------+--------------+------+-----+---------+----------------+
```

| Field | Note  |
|---|---|
| id | This is the primary key for this table.  |
| group_id | This is the id for the group table entry that represents the destination app. |
| port | We used to only allow a single port per network policy. Now we allow a range of ports. This value is no longer used. Instead the "start_port" and "end_port" values are used. |
| protocol | This is the protocol (udp, icmp, tcp, all) allowed by the network policy. |
| start_port | This is the start of the port range for the network policy. |
| end_port | This is the end of the port range for the network policy. |


### <a name="policies-table"></a> Policies
There is an entry in the policies table added for each network policy created.

```
mysql> describe policies;
+----------------+---------+------+-----+---------+----------------+
| Field          | Type    | Null | Key | Default | Extra          |
+----------------+---------+------+-----+---------+----------------+
| id             | int(11) | NO   | PRI | NULL    | auto_increment |
| group_id       | int(11) | YES  | MUL | NULL    |                |
| destination_id | int(11) | YES  |     | NULL    |                |
+----------------+---------+------+-----+---------+----------------+
```

| Field | Note  |
|---|---|
| id | This is the primary key for this table.  |
| group_id | This is the id for the group table entry that represents the source app. |
| destination_id | This is the id for the destinations table entry that represents the destination metadata. |


## <a name="network-policy-example"></a> Networking Policy Example

In this example: 
* There is a network policy from AppA to AppB. 
* AppA has guid `2ffe4b0f-b03c-48bb-a4fa-bf22657d34a2`
* AppB has guid `5346072e-7265-45f9-b70a-80c42e3f13ae`


```
mysql> select * from policies;           mysql> select * from destinations;
+---------------+----------------+       +---------------+------+----------+------------+----------+
| id | group_id | destination_id |       | id | group_id | port | protocol | start_port | end_port |
+--------------------------------+       +---------------------------------------------------------+
|  1 |        1 |              1 |    +-->  1 |        2 | 8080 | tcp      |       8080 |     8080 |
+----+--------+-+--------------+-+    |  +----+--------+-+------+----------+------------+----------+
              |                |      |                |
              |                +------+                +---------------+
              |                                                        |
              |                                                        |
              |                                                        |
              |  mysql> select * from groups limit 3;                  |
              |  +-------------------------------------------+------+  |
              |  | id | guid                                 | type |  |
              |  +--------------------------------------------------+  |
              +-->  1 | 2ffe4b0f-b03c-48bb-a4fa-bf22657d34a2 | app  |  |
                 |  2 | 5346072e-7265-45f9-b70a-80c42e3f13ae | app  <--+
                 |  3 | NULL                                 | app  |
                 +----+--------------------------------------+------+
```

## <a name="security-groups-tables"</a> Security Group Related Tables

There are three tables storing information about security groups: security_groups, running_security_groups_spaces,
and staging_security_groups_spaces.


### <a name="security-groups-table"></a> security_groups
This table stores a copy of all security groups found in CAPI, so vxlan-policy-agent can query
policy-server-internal for this information, rather than overwhelm CAPI with requests. Its data is
synced and updated via the policy-server-asg-syncer process, and is not a source of truth for ASG data.

```
mysql> describe security_groups;
+-----------------+--------------+------+-----+---------+----------------+
| Field           | Type         | Null | Key | Default | Extra          |
+-----------------+--------------+------+-----+---------+----------------+
| id              | bigint       | NO   | PRI | NULL    | auto_increment |
| guid            | varchar(36)  | NO   | UNI | NULL    |                |
| name            | varchar(255) | NO   |     | NULL    |                |
| rules           | mediumtext   | YES  |     | NULL    |                |
| staging_default | tinyint(1)   | YES  | MUL | 0       |                |
| running_default | tinyint(1)   | YES  | MUL | 0       |                |
| staging_spaces  | json         | YES  |     | NULL    |                |
| running_spaces  | json         | YES  |     | NULL    |                |
+-----------------+--------------+------+-----+---------+----------------+
```

| Field | Note |
|---|---|
| id | An internal id for each record |
| guid | The CAPI GUID of the security group |
| name | The name of the security group as it appears in CAPI |
| hash | A SHA256 hash of the ASG data, used to check whether records need updating during policy-server-asg-syncer polls |
| rules | The rules associated with the ASG defined in CAPI |
| staging_default | Whether or not this is a globally bound security group for `staging` lifecycles |
| running_default | Whether or not this is a globally bound security group for `running` lifecycles |
| staging_spaces | A json list of all spaces this security group is bound to for the `staging` lifecycle |
| running_spaces | A json list of all spaces this security group is bound to for the `running` lifecycle |

### <a name="running-security-groups-spaces-table"></a> running_security_groups_spaces
This table is a join table to enable faster querying of security_groups when filtering by
running_space guids. It is synced and updated via the policy-server-asg-syncer process, and is not a source of
truth for ASG data.

```
mysql> describe running_security_groups_spaces;
+---------------------+--------------+------+-----+---------+-------+
| Field               | Type         | Null | Key | Default | Extra |
+---------------------+--------------+------+-----+---------+-------+
| space_guid          | varchar(255) | NO   | PRI | NULL    |       |
| security_group_guid | varchar(255) | NO   | PRI | NULL    |       |
+---------------------+--------------+------+-----+---------+-------+
```

| Field | Note|
|---|---|
| space_guid | This value is the CAPI guid for the space bound to a given security group via the `running` app lifecycle |
| security_group_guid | This value is the CAPI guid for the security group bound to a given space via the `running` app lifecycle |


### <a name="staging-security-groups-spaces-table"></a> staging_security_groups_spaces
This table is a join table to enable faster querying of security_groups when filtering by
staging_space guids. It is synced and updated via the policy-server-asg-syncer process, and is not a source of
truth for ASG data.

```
mysql> describe staging_security_groups_spaces;
+---------------------+--------------+------+-----+---------+-------+
| Field               | Type         | Null | Key | Default | Extra |
+---------------------+--------------+------+-----+---------+-------+
| space_guid          | varchar(255) | NO   | PRI | NULL    |       |
| security_group_guid | varchar(255) | NO   | PRI | NULL    |       |
+---------------------+--------------+------+-----+---------+-------+
```

| Field | Note|
|---|---|
| space_guid | This value is the CAPI guid for the space bound to a given security group via the `staging` app lifecycle |
| security_group_guid | This value is the CAPI guid for the security group bound to a given space via the `staging` app lifecycle |

## <a name="migrations-tables"></a> Migration Related Tables

There are two tables related to migrations: gorp_migrations and gorp_lock. 

### <a name="gorp-mirations-table"></a> gorp_migrations
This table tracks what database migrations have been applied.

```
mysql> describe gorp_migrations;
+------------+--------------+------+-----+---------+-------+
| Field      | Type         | Null | Key | Default | Extra |
+------------+--------------+------+-----+---------+-------+
| id         | varchar(255) | NO   | PRI | NULL    |       |
| applied_at | datetime     | YES  |     | NULL    |       |
+------------+--------------+------+-----+---------+-------+
```

| Field | Note  |
|---|---|
| id | This value refers to the "id" of the migration as specified in [migrations.go](https://github.com/cloudfoundry/cf-networking-release/blob/develop/src/code.cloudfoundry.org/policy-server/store/migrations/migrations.go) |
| applied_at | This is the time when the migration was applied to the database. |

### <a name="gorp-lock-table"></a> gorp_lock
There can be many VMs running a policy-server process, however only one process needs to run the migrations. Before one policy-server runs the migrations it grabs the lock using this table. Other policy-server instances will see that the lock is taken and will not attempt to do migrations at the same time.
Unless a migration is running this table will be empty.

```
mysql> describe gorp_lock;
+-------------+--------------+------+-----+---------+-------+
| Field       | Type         | Null | Key | Default | Extra |
+-------------+--------------+------+-----+---------+-------+
| lock        | varchar(255) | NO   | PRI | NULL    |       |
| acquired_at | datetime     | YES  |     | NULL    |       |
+-------------+--------------+------+-----+---------+-------+
```

| Field | Note  |
|---|---|
| lock | A value representing the policy-server that is currently running a migration.|
| applied_at | The time that the policy-server claimed the lock. |

