## Expectations for CNI plugin developers
### MTU
CNI plugins should automatically detect the MTU settings on the host, and set the MTU
on container network interfaces appropriately.  For example, if the host MTU is 1500 bytes
and the plugin encapsulates with 50 bytes of header, the plugin should ensure that the
container MTU is no greater than 1450 bytes.  This is to ensure there is no fragmentation.
The built-in flannel CNI plugin does this.

## To replace flannel with your own CNI plugin
0. Remove the following BOSH jobs:
  - `cni-flannel`
  - `vxlan-policy-agent`
0. Remove the following BOSH packages:
  - `flannel`
  - `flannel-watchdog`
0. Add in all packages and jobs required by your CNI plugin.  At a minimum, you must provide a CNI binary program and a CNI config file.
  - For more info on **bosh packaging scripts** read [this](http://bosh.io/docs/packages.html#create-a-packaging-script).
  - For more info on **bosh jobs** read [this](http://bosh.io/docs/jobs.html).
0. Update the [deployment manifest properties](http://bosh.io/docs/deployment-manifest.html#properties)

  ```yaml
  garden-cni:
    adapter:
      cni_plugin_dir: /var/vcap/packages/YOUR_PACKAGE/bin # your CNI binary goes in this directory
      cni_config_dir: /var/vcap/jobs/YOUR_JOB/config/cni  # your CNI config file goes in this directory
  ```
  Remove any lingering references to `flannel` or `cni-flannel` in the deployment manifest.

## To deploy a local-only (no-op) CNI plugin
As a baseline, you can deploy using only the basic [bridge CNI plugin](https://github.com/containernetworking/cni/blob/master/Documentation/bridge.md).

This plugin will provide connectivity between containers on the same Garden host (Diego cell)
but will not provide a cross-host network.  However, it can be a useful baseline configuration for
testing and development.

```bash
cd bosh-lite
bosh target lite
bosh update cloud-config cloud-config.yml
bosh deployment local-only.yml
bosh deploy
```

## To deploy diego with CNI but without cross-host container networking
For generating a cloudfoundry-diego deployment without container to container connectivity, but using the CNI bridge plugin for NAT'ed connectivity.

```bash
CNI_BRIDGE=true ./scripts/generate-bosh-lite-manifests
bosh deploy
```
