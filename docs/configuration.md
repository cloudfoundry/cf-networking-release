# Configuration Information for Operators

## Table of Contents
0. [Network Policy Access Control](#network-policy-access-control)
0. [Database Configuration](#database-configuration)
0. [Mutual TLS](#mutual-tls)
0. [Max Open/Idle Connections](#max-openidle-connections)

## Network Policy Access Control

#### Network Admin Access

Any user with the `network.admin` UAA scope may create create network policies
between any two applications.  There is no limit on the number of policies a
network admin can configure.

#### App Developer Access
Application developers may be given a reduced set of permissions for configuring
network policy.  In this permission model a user may configure policies between
apps that are in spaces in which this user has the `SpaceDeveloper` role in
CloudController.  An application may be the source of only a limited number of
policies created this way (the limit is configurable via the BOSH property
`cf_networking.max_policies_per_app_source`, defaults to 50).

- To grant an individual user this access, give them the `network.write` scope
  in UAA
- To grant **all** users this level of access, set the BOSH property
  `cf_networking.enable_space_developer_self_service` to `true`


## Database Configuration
A SQL database is required to store Network Policies.  MySQL and PostgreSQL
databases are currently supported.

### Hosting options
The database may be hosted anywhere that the Policy Server BOSH VM can reach it,
including on another BOSH-deployed VM or on a cloud-provided service.  Here are
some options:

#### MySQL

- Add a logical database to the CF-MySQL cluster that ships with
  [CF-Deployment](https://github.com/cloudfoundry/cf-deployment).

- BOSH-deploy the [CF-MySQL
  release](https://github.com/cloudfoundry/cf-mysql-release) to dedicated VM(s).
  CF-MySQL may be deployed either as a single-node or as a highly available (HA)
  cluster.

- Use a database service provided by your cloud infrastructure provider.  For
  example, in some of our automated tests we use an AWS RDS MySQL instance
  configured as follows:

    - MySQL 5.7.16
    - db.t2.medium (4 Gib)
    - 20 GB storage


#### PostgreSQL

- Use a database service provided by your cloud infrastructure provider.  For
  example, in some of our automated tests we use an AWS RDS PostgreSQL instance
  configured as follows:

  - PostgreSQL 9.5.4
  - db.m3.medium (3.75 GiB)
  - 20 GB storage

- BOSH-deploy the [Postgres
  release](https://github.com/cloudfoundry/postgres-release/) to a dedicated VM.

### Policy Server DB scale and performance testing

Policy server performance has been validated for deployments with:

  - 100 cells
  - 20k applications
  - 1 instance per app
  - 60k policies
  - 20 requests per second

To reach these numbers we deployed:

  - 2 policy server instances (t2.large on AWS with 10GB ephemeral disk)
  - 1 CF MySQL instance (r3.4xlarge on AWS)

The bottleneck for performance seems to usually be the VM hosting the database.
If you are scaling above 30k policies, we suggest deploying the VM hosting the
database with a r3.4xlarge, a memory-intensive instance-type, if you are on AWS.

We recommend having at least 2 instances of the policy server for high
availability. We saw little to no performance gain with 4 instances of the
policy server for the above scaling tests.

## Mutual TLS

There is a control-plane connection between the following system components:

- The VXLAN Policy Agent (in [silk-release](https://github.com/cloudfoundry/silk-release))
  is a client of the internal Policy Server API

This connection requires Mutual TLS.

If you want to generate them yourself, ensure that all certificates support the
cipher suite `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`.  The Policy Server will
reject connections using any other cipher suite.

## Max Open/Idle Connections

In order to limit the number of open or idle connections between the
policy-server and database, the following properties can be set.  On both the
`policy-server` and `policy-server-internal` jobs.
- `max_open_connections`
- `max_idle_connections`

By default there is no limit to the number of open or idle connections.
