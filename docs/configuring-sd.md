## Internal Domains

### Configuring Custom Internal Domains

Creating your own internal domain requires [enable-service-discovery
opsfile](https://github.com/cloudfoundry/cf-deployment/blob/master/operations/enable-service-discovery.yml)
and the following two operations:
1. Add the custom internal domain name(s) to the `internal_domains` property on
   the `bosh-dns-adapter` job.

```yaml
- type: replace
  path: /instance_groups/name=diego-cell/jobs/name=bosh-dns-adapter/properties/internal_domains?
  value: ["apps.internal."]
```

> NOTE: The internal domain property in bosh-dns-adapter supports domains with
> and without the trailing dot.

2. Run the following command after deployment:

```bash
cf create-shared-domain <DOMAIN> --internal
```

Or, add the custom internal domain to the `apps_domains` property on
`cloud_controller_ng` job.

```yaml
- type: replace
  path: /instance_groups/name=api/jobs/name=cloud_controller_ng/properties/app_domains/-
  value:
    name: apps.internal
    internal: true
```

NOTE: The internal domain property in cloud_controller_ng does not accept
domains with a trailing dot.

3. Deploy.

To delete a shared domain, run one of the following commands:

```bash
cf curl -X DELETE /v2/shared_domains/<SHARED DOMAIN GUID>
```

```bash
cf delete-shared-domain <DOMAIN> [-f]
```