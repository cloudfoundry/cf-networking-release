# Network Policy Database Overview

This document is intended to help people who are poking around the `network_policy` database. There are a lot of tables (most of them empty) and the names of the rows are not the most intuative. This doc should help you connect to the database and understand what you find there.

## Table of Contents

* [How to access an internal database](#access-db)
* [Table Overview](#table-overview)
* [Networking Policy Related Tables](#network-policy-tables)
  * [Groups](#groups-table)
  * [Destinations](#destinations-table)
  * [Policies](#policies-table)
* [Networking Policy Example Rows](#network-policy-example)
* [Migration Related Tables](#migration-tables)
  * [Gorp_migrations](#gorp-migrations-table)
  * [Gorp_lock](#gorp-lock-table)
* [Dynamic Egress Related Tables](#dynamic-egress-tables)

-------------------------------------------------------------------------------------------

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
| apps | Related to dyanmic egress which is deprecated. Should be empty. |
| defaults |  Related to dyanmic egress which is deprecated. Should be empty. |
| destination_metadatas  |  Related to dyanmic egress which is deprecated. Should be empty.  |
| destinations | List of metadata about network policies. |
| egress_policies  |  Related to dyanmic egress which is deprecated. Should be empty. |
| gorp_lock  | Locking mechanism for running migrations. |
| gorp_migrations  | Record of which migrations have been run. |
| groups  | List of all apps that are either the source or destination of a network policy. |
| ip_ranges  | Related to dyanmic egress which is deprecated. Should be empty. |
| policies  | List of source apps and destination metadata for network policies. |
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


## <a name="migrations-tables"></a> Migration Related Tables

There are two tables related to migraitons: gorp_migrations and gorp_lock. 

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
| id | This value refers to the "id" of the migration as specified in [migrations.go](https://github.com/cloudfoundry/cf-networking-release/blob/develop/src/policy-server/store/migrations/migrations.go) |
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


## <a name="dynamic-egress-tables"></a> Dynamic Egress Related Tables

There are 7 tables related to dynamic egress policies: apps, defaults, destination_metadatas, egress_policies, ip_ranges, terminals, and spaces. 

Dynamic egress was a beta feature that we are no longer planning on taking GA. These tables should be empty.



