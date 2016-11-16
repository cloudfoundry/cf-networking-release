## Manifest property changes

See [AWS](aws.md) deployment docs for examples

###  0.7.0

Netman stub

```diff
        policy-server:
          uaa_client_secret: REPLACE_WITH_UAA_CLIENT_SECRET
          uaa_url: (( "https://uaa." config_from_cf.system_domain ))
+         cc_url: (( "https://api." config_from_cf.system_domain ))
          skip_ssl_validation: true
```

CF stub

```diff
     network-policy:
-      authorities: uaa.resource
+      authorities: uaa.resource,cloud_controller.admin_read_only
+      authorized-grant-types: client_credentials,refresh_token
       secret: REPLACE_WITH_UAA_CLIENT_SECRET
```
