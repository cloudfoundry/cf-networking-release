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
| enable-space-developer-self-service.yml | | |
| increase-diego-cell-mtu.yml |  |  |
| large-disk.yml |  |  |
| lower-canary-watch-time.yml |  |  |
| scale-diego-cell-max-in-flight.yml |  |  |
| scale-diego-cell-vm-size.yml |  |  |  
| scale-ephemeral-disk.yml |  |  |
| scale-instances-to-2.yml |  |  |
| scale-max-containers.yml |  |  |
| scale-persistent-disk.yml |  |  |
| scale-router-ephemeral-disk.yml |  |  |
| scale-to-2-diego-cells.yml |  |  |
| scale-to-3-diego-cells.yml |  |  |
| scale-up-4-api-instances.yml |  |  |
| smaller-footprint.yml |  |  |
| use-external-postgres-for-c2c.yml |  |  |
| use-latest-capi.yml |  |  |
| use-latest-cf-networking.yml |  |  |
| use-latest-silk.yml |  |  |
|  |  |  |

