# 3rd Party Plugin Development for Container Networking

## Notes for CNI plugin developers

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

## What data will my CNI plugin receive?
The `garden-external-networker` will invoke one or more CNI plugins, according to the [CNI Spec](https://github.com/containernetworking/cni/blob/master/SPEC.md).
It will start with the CNI config files available in the `cni_config_dir` and also inject
some dynamic information about the container, including the CloudFoundry App, Space and Org that it belongs to.

The Network Configuration data that is passed to the `wrapper` plugin is generated from this [template](../jobs/cni-flannel/templates/30-cni-wrapper-plugin.conf.erb).

Here's an example:
```json
{
  {
    "name": "cni-wrapper",
    "type": "wrapper",
    "cniVersion": "0.2.0",
    "datastore": "/path/to/datastore",
    "delegate": {
      "name": "cni-flannel",
      "type": "flannel",
      "subnetFile": "/var/vcap/data/flannel/subnet.env",
      "dataDir": "/var/vcap/data/flannel/data",
      "delegate": {
        "bridge": "cni-flannel0",
        "isDefaultGateway": true,
        "ipMasq": false
       }
    }
  }
  "metadata": {
    "app_id": "d5bbc5ed-886a-44e6-945d-67df1013fa16",
    "org_id": "2ac41bbf-8eae-4f28-abab-51ca38dea3e4",
    "policy_group_id": "d5bbc5ed-886a-44e6-945d-67df1013fa16",
    "space_id": "4246c57d-aefc-49cc-afe0-5f734e2656e8"
  }
}
```
Note that the `delegate`, `name` and `type` fields are present in the static `30-cni-wrapper-plugin.conf` file provided by the BOSH release.
At runtime, the `garden-external-networker` also injects the `network` field with `properties` which include CF-specific info.

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


## Policy Server Internal API
To replace the VXLAN Policy Agent with your own Policy Enforcement implementation,
you can use the Policy Server's internal API to retrieve policy information.

There is a single endpoint to retrieve policies:

`GET https://policy-server.service.cf.internal:4003/networking/v0/internal/policies`

Additionally, you can use the `id` query parameter to filter the response to include
only policies with a source or destination that match any of the comma-separated
`group_policy_id`'s that are included.

### TLS configuration
The Policy Server internal API requires Mutual TLS.  All connections must use a client certificate
that is signed by a trusted certificate authority.

This CA is configured for the policy server in the bosh deployment manifest
property `properties.policy-server.ca_cert`.

An example can be found in the `bosh-lite` stubs included in this repository
[here](../bosh-lite/deployments/diego_with_netman.yml).

Additionally, the server requires that connections use the TLS cipher suite
`TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`.  Your client must support this cipher suite.

We provide [a script](../scripts/generate-certs) to generate all required certs & keys.

### Policy Server Internal API Details

`GET /networking/v0/internal/policies`

List all policies optionally filtered to match requested  `policy_group_id`'s

Query Parameters:

- `id`: comma-separated `policy_group_id` values

Response Body:

- `policies`: list of policies
- `policies[].destination`: the destination of the policy
- `policies[].destination.id`: the `policy_group_id` of the destination (currently always an `app_id`)
- `policies[].destination.port`: the `port` allowed on the destination
- `policies[].destination.protocol`: the `protocol` allowed on the destination: `tcp` or `udp`
- `policies[].destination.tag`: the `tag` of the source allowed to the destination
- `policies[].source`: the source of the policy
- `policies[].source.id`: the `policy_group_id` of the source (currently always an `app_id`)
- `policies[].source.tag`: the `tag` of the source allowed to the destination

### Examples Requests and Responses

#### Get all policies

```bash
curl -s \
  --cacert certs/ca.crt \
  --cert certs/client.crt \
  --key certs/client.key \
  https://policy-server.service.cf.internal:4003/networking/v0/internal/policies
```

```json
  {
      "policies": [
        {
            "destination": {
                "id": "eb95ff20-cba8-4edc-8f4a-cf80d0669faf",
                "port": 8080,
                "protocol": "tcp",
                "tag": "0002"
            },
            "source": {
                "id": "4a2d3627-0b8c-42d1-9563-22696eedc05d",
                "tag": "0001"
            }
        },
        {
            "destination": {
                "id": "b611f7e6-c8fe-41cb-b150-92581aafa5c2",
                "port": 8080,
                "protocol": "tcp",
                "tag": "0004"
            },
            "source": {
                "id": "3b348978-a3cb-487c-a277-58fdc3e2c678",
                "tag": "0003"
            }
        },
        {
            "destination": {
                "id": "8fa287c9-0d01-491e-a1d5-d6e2d8a1ef61",
                "port": 8080,
                "protocol": "tcp",
                "tag": "0005"
            },
            "source": {
                "id": "8fa287c9-0d01-491e-a1d5-d6e2d8a1ef61",
                "tag": "0005"
            }
        },
        {
            "destination": {
                "id": "d5bbc5ed-886a-44e6-945d-67df1013fa16",
                "port": 5555,
                "protocol": "tcp",
                "tag": "0006"
            },
            "source": {
                "id": "d5bbc5ed-886a-44e6-945d-67df1013fa16",
                "tag": "0006"
            }
        }
    ]
}
```

#### Get filtered policies

Returns all policies with source or destination id's that match any of the
included `policy_group_id`'s.

```bash
curl -s \
--cacert certs/ca.crt \
--cert certs/client.crt \
--key certs/client.key \
https://policy-server.service.cf.internal:4003/networking/v0/internal/policies?id=5351a742-6704-46df-8de0-1a376adab65c,d5bbc5ed-886a-44e6-945d-67df1013fa16
```

```json
{
    "policies": [
        {
            "destination": {
                "id": "d5bbc5ed-886a-44e6-945d-67df1013fa16",
                "port": 5555,
                "protocol": "tcp",
                "tag": "0006"
            },
            "source": {
                "id": "d5bbc5ed-886a-44e6-945d-67df1013fa16",
                "tag": "0006"
            }
        },
        {
            "destination": {
                "id": "5351a742-6704-46df-8de0-1a376adab65c",
                "port": 5555,
                "protocol": "tcp",
                "tag": "0007"
            },
            "source": {
                "id": "5351a742-6704-46df-8de0-1a376adab65c",
                "tag": "0007"
            }
        }
    ]
}
```
