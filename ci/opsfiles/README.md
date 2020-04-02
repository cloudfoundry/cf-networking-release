# Ops-files

This is the README for our Opsfiles. To learn more about `cf-networking-release`, go to the main [README](../../README.md).

| Name | Purpose | Notes |
| --- | --- | --- |
| add-acceptance-test-jobs.yml | Add acceptance test jobs: iptables-writer, test-server. | Used in Pickelhelm and Mitre in CI. |
| add-smoke-test-user.yml | Adds a new user called "cf-networking-smokes" with groups needed for smoke tests. | Used in Mitre in CI. |
| add-temporary-istio-apps-internal.yml | Adds "istio.apps.internal" as an istio internal domain. | No one is using. We should delete. |
| change-nats-monitor-port.yml | Changes the monitor port to 8222 for the nats job. | Used in Toque in CI. |
| change-vtep-port.yml | Changes the vtep port to 4800 for the silk-daemon job. | Used in Mitre in CI. |
| custom-apps-domain.yml | Replaces the apps domain with one specified credhub. | Used in Mitre in CI. This will only have the desired affect on the initial deploy. Changing it on an existing deployment will only append the domain, not replace old ones. |
| datadog.yml | Adds configuration for datadog-firehose-nozzle to the uaa job. | Don't think anyone is using. Should delete. Was used in istio-release pipeline.|
| diego-instance-identity.yml | ??? | Don't think anyone us using. Should delete. |
| disable-ingress-redirect-to-proxy.yml	 | This distables the iptables rule that sends all app traffic to envoy. This "feature" broke tcp routing. | We shouldn't be deploying istio-release in CI, but we are so this is currently needed. We should stop and get rid of this opsfile too. |
| enable-rate-limiting-for-udp.yml | Increases rate limiting for iptables logging UDP traffic. Set to allow only one log per second. | Used in Mitre in CI.|
| enable-space-developer-self-service.yml | Sets the `enable-space-developer-self-service` property on the policy-server to true. This allows space.devs to create network policies for their apps. | Used in Mitre in CI. |
| increase-diego-cell-mtu.yml | Sets the `mtu` property on the silk-cni job to 1350. | Used by Toque in CI. |
| large-disk.yml | Sets the disk_size to 65536. | Not used by any environments in CI. Maybe we can delete? |
| lower-canary-watch-time.yml | Sets the canary watch time to 5000-1200000. | Used by Pickelhelm, Mitre, Toque, and Caubeen in CI. |
| scale-diego-cell-max-in-flight.yml | Sets the max_in_flight for diego cell instance groups to 20.  | Used by Toque in CI. Toque only has three Diego Cells, so I'm not sure why this is necessary. |
| scale-diego-cell-vm-size.yml | Sets the Diego Cell VM type to "n1-highmem-4". | Used by Toque in CI. |
| scale-ephemeral-disk.yml | Sets the scheduler instance group to have a 100GB ephemeral disk. | Used by Mitre in CI. |
| scale-instances-to-2.yml | Sets the following instance groups to have two instances: api, cc-worker, diego-api, diego-cell, nats, router, and uaa. | Used by Mitre in CI. |
| scale-max-containers.yml | Sets the max_containers property on the garden job to 1000. This allows garden to create 1000 containers at a time per Diego Cell. | Used by Toque in CI. |
| scale-persistent-disk.yml | Sets the size of the persistent disk on the database VM to 100GB. | Used by Mitre in CI. |
| scale-router-ephemeral-disk.yml | Sets the size of the ephemeral disk on the router vm to 10GB. | Used by Toque in CI. |
| scale-to-2-diego-cells.yml | Sets the number of instances of Diego Cells to 2. | Used by Pickelhelm in CI. |
| scale-to-3-diego-cells.yml | Sets the number of instances of Diego Cells to 3. | Used by Toque in CI. |
| scale-up-4-api-instances.yml | Sets the number of instances of api VMs to 4. | Used by Caubeen in CI. |
| smaller-footprint.yml | Sets the VM type for many instances groups to be n1-standard-1. | Used by Pickelhelm, Mitre, and Toque in CI. |
| use-latest-capi.yml | Uses the latest available capi-release. | Used in Pickelhelm, Mitre, and Toque in CI. This was originally done when we were developing a feature with the CAPI team and needed their latest changes to make our pipeline green. We no longer need to use this opsfile in CI. |
| use-latest-cf-networking.yml | Uses the latest available cf-networking-release. | Used by Pickelhelm, Mitre, and Toque in CI. |
| use-latest-silk.yml | Uses the latest available silk-release. | Used by Pickelhelm, Mitre, Toque, and Caubeen in CI. |

