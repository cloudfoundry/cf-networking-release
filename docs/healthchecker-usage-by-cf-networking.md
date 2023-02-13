# HealthChecker usage by Cf-networking-release

A few of the components within cf-networking-release utilize the <a href="https://github.com/cloudfoundry/healthchecker-release">Healthchecker-release</a> to perform it's monit healthchecks.  The healthchecker add TCP and HTTP healtchecks to extend standard monit service checks.  Since the version of monit included in BOSH does not support specific tcp/http health checks, we designed this utility to perform health checking and restart processes if they become unreachable.

## Components that utlize the healthchecker-release

- bosh-dns-adapter
- silk-daemon 

## How it Works
Healthchecker is added to a bosh release as a monit process under the Job that is to be monitored. It is configured to perform a healthcheck against the main process in the Job. If healthchecker detects a failure, it will panic and exit. The healthchecker supplementary script restarts the main monit process, allowing up to ten failures in a row. After 10 consecutive failures, it gives up, since restarting the process is either in a poor state, or the healthchecker is misconfigured and should not be causing process downtime.

This component typically requires no additional configuration from platform operators.

