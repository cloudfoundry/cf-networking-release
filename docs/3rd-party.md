# 3rd Party Plugin Development for Container Networking

*If you want to integrate your own CNI plugin with Cloud Foundry, review the component diagrams on the [architecture page](arch.md) and read this document.*

If you want to integrate using the default values for the `cni_config_dir` and `cni_plugin_dir`, the package for the CNI plugin *must* be named `cni`
and the job for the CNI plugin *must* be named `cni`.

If you have any questions or feedback, please visit the `#container-networking` channel on [Cloud Foundry Slack](http://slack.cloudfoundry.org/).

## Introduction
Basic network connectivity is configured according to the [CNI specification](https://github.com/containernetworking/cni/blob/master/SPEC.md).

However, Cloud Foundry requires the networking stack to perform certain additional functions which are currently not standardized by CNI.  These are:

1. Expose [container ports on the diego cell via DNAT](https://docs.run.pivotal.io/devguide/deploy-apps/environment-variable.html#CF-INSTANCE-PORTS)

2. Enforce [Cloud Foundry Application Security Groups](https://docs.cloudfoundry.org/concepts/asg.html) for egress traffic from the application container 

3. Enforce Container to Container Network Policies that have been configured in the [Policy Server API](API.md)

Configuration for (1) and (2) is passed down via the semi-standardized `runtimeConfig` field described in the [CNI convensions document](https://github.com/containernetworking/cni/blob/master/CONVENTIONS.md).  See [What data will my CNI plugin receive](#what-data-will-my-cni-plugin-receive) below.

Configuration for (3) is available via the [Policy Server Internal API](#policy-server-internal-api).

## Test suites
A Cloud Foundry system that integrates a 3rd party networking component should be able to pass the following test suites:

- [CF Networking Smoke Tests](../src/test/smoke)
- [CF Networking Acceptance Tests](../src/test/acceptance)
- [CF Acceptance Tests (CATs)](https://github.com/cloudfoundry/cf-acceptance-tests/)

The smoke tests are non-disruptive and may be run against a live, production environment.  The other tests make potentially disruptive changes and should only be run against a non-production environment.

For local development, we recommend using [`cf-deployment` on BOSH-lite](https://github.com/cloudfoundry/cf-deployment).

For guidance on these test suites, please reach out to our team in Slack (top of this page).

## MTU
CNI plugins should automatically detect the MTU settings on the host, and set the MTU
on container network interfaces appropriately.  For example, if the host MTU is 1500 bytes
and the plugin encapsulates with 50 bytes of header, the plugin should ensure that the
container MTU is no greater than 1450 bytes.  This is to ensure there is no fragmentation.
The built-in silk CNI plugin does this.

Operators may wish to override the MTU setting. In this case they will set the BOSH property `cf_networking.mtu`.
3rd party plugins should respect this value.

## To author a BOSH release with your plugin
0. Remove the following BOSH jobs:
  - `silk-cni`
  - `silk-daemon`
  - `silk-controller`
  - `vxlan-policy-agent`
0. Remove the following BOSH packages:
  - `silk-cni`
  - `silk-daemon`
  - `silk-controller`
  - `vxlan-policy-agent`
0. Add in all packages and jobs required by your CNI plugin.  At a minimum, you must provide a CNI binary program and a CNI config file.
   If your software requires a long-lived daemon to run on the diego cell, we recommend you deploy a separate BOSH job for that.
  - For more info on **bosh packaging scripts** read [this](http://bosh.io/docs/packages.html#create-a-packaging-script).
  - For more info on **bosh jobs** read [this](http://bosh.io/docs/jobs.html).


## To deploy your BOSH release with Cloud Foundry
0. Update the [deployment manifest properties](http://bosh.io/docs/deployment-manifest.html#properties)

  ```yaml
  properties:
    cf_networking:
      cni_plugin_dir: /var/vcap/packages/YOUR_PACKAGE/bin # directory for CNI binaries
      cni_config_dir: /var/vcap/jobs/YOUR_JOB/config/cni  # directory for CNI config file(s)
  ```

  Remove any lingering references to `vxlan-policy-agent` and `silk-*` in the deployment manifest, and replace the `plugin` properties
  with any manifest properties that your bosh job requires.


## What data will my CNI plugin receive?
The `garden-external-networker` will invoke one or more CNI plugins, according to the [CNI Spec](https://github.com/containernetworking/cni/blob/master/SPEC.md).
It will start with the CNI config files available in the `cf_networking.cni_config_dir` and also inject
some dynamic information about the container. This is divided into two keys the first, `metadata`
contains the CloudFoundry App, Space and Org that it belongs to. Another key `runtimeConfig` holds information that CNI plugins may need
to implement legacy networking features of Cloud Foundry. It is divided into two keys, `portMappings` can be translated into port forwarding
rules to allow the gorouter access to application containers, and `netOutRules` which are egress whitelist rules used for implementing
application security groups.

A reference implementation of these features can be seen in in the [cni-wrapper-plugin](../src/cni-wrapper-plugin).

At deploy time, Silk's CNI config is generated from this [template](../jobs/silk-cni/templates/cni-wrapper-plugin.conf.erb), and
is stored in a file on disk at `/var/vcap/jobs/silk-cni/config/cni-wrapper-plugin.conf`, which resembles

```json
{
  "name": "cni-wrapper",
  "type": "cni-wrapper-plugin",
  "cniVersion": "0.3.1",
  "datastore": "/var/vcap/data/container-metadata/store.json",
  "iptables_lock_file": "/var/vcap/data/garden-cni/iptables.lock",
  "overlay_network": "10.255.0.0/16",
  "instance_address": "10.0.16.14",
  "iptables_asg_logging": true,
  "iptables_c2c_logging": true,
  "ingress_tag": "ffff0000",
  "dns_servers": [

  ],
  "delegate": {
    "cniVersion": "0.3.1",
    "name": "silk",
    "type": "silk-cni",
    "daemonPort": 23954,
    "dataDir": "/var/vcap/data/host-local",
    "datastore": "/var/vcap/data/silk/store.json",
    "mtu": 0
  }
}
```

Then, when a container is created, the `garden-external-networker` adds additional runtime-specific data, so that
the CNI plugin receives a final config object that resembles:

```json
{
  "name": "cni-wrapper",
  "type": "cni-wrapper-plugin",
  "cniVersion": "0.3.1",
  "datastore": "/var/vcap/data/container-metadata/store.json",
  "iptables_lock_file": "/var/vcap/data/garden-cni/iptables.lock",
  "overlay_network": "10.255.0.0/16",
  "instance_address": "10.0.16.14",
  "iptables_asg_logging": true,
  "iptables_c2c_logging": true,
  "ingress_tag": "ffff0000",
  "dns_servers": [

  ],
  "delegate": {
    "cniVersion": "0.3.1",
    "name": "silk",
    "type": "silk-cni",
    "daemonPort": 23954,
    "dataDir": "/var/vcap/data/host-local",
    "datastore": "/var/vcap/data/silk/store.json",
    "mtu": 0
  },
  "runtimeConfig": {
    "portMappings": [{
      "host_port": 60001,
      "container_port": 8080
    }, {
      "host_port": 60002,
      "container_port": 2222
    }],
    "netOutRules": [{
      "protocol": 1,
      "networks": [{
        "start": "8.8.8.8",
        "end": "9.9.9.9"
      }],
      "ports": [{
        "start": 53,
        "end": 54
      }],
      "log": true
    }],
    "metadata": {
      "policy_group_id": "d5bbc5ed-886a-44e6-945d-67df1013fa16",
      "app_id": "d5bbc5ed-886a-44e6-945d-67df1013fa16",
      "space_id": "4246c57d-aefc-49cc-afe0-5f734e2656e8",
      "org_id": "2ac41bbf-8eae-4f28-abab-51ca38dea3e4"
    }
  }
}
```

Furthermore, the CNI runtime data, provided as environment variables, sets the
[CNI `ContainerID`](https://github.com/containernetworking/cni/blob/master/SPEC.md#parameters) equal to the
[Garden container `Handle`](https://godoc.org/code.cloudfoundry.org/garden#ContainerSpec).

When [Diego](https://github.com/cloudfoundry/diego-release) calls Garden, it sets that equal to the [`ActualLRP` `InstanceGuid`](https://godoc.org/code.cloudfoundry.org/bbs/models#ActualLRPInstanceKey).
In this way, a 3rd-party system can relate data from CNI with data in the [Diego BBS](https://github.com/cloudfoundry/bbs/tree/master/doc).



## Policy Server Internal API
If you are replacing the built-in "VXLAN Policy Agent" with your own Policy Enforcement implementation,
you can use the Policy Server's internal API to retrieve policy information.

There is a single endpoint to retrieve policies:

`GET https://policy-server.service.cf.internal:4003/networking/v1/internal/policies`

Additionally, you can use the `id` query parameter to filter the response to include
only policies with a source or destination that match any of the comma-separated
`group_policy_id`'s that are included.

### TLS configuration
The Policy Server internal API requires Mutual TLS.  All connections must use a client certificate
that is signed by a trusted certificate authority.  The certs and keys should be configured via BOSH manifest
properties on the Policy Server and on your custom policy client, e.g.

```yaml
properties:
  cf_networking:
    policy_server:
      ca_cert: |
        -----BEGIN CERTIFICATE-----
        REPLACE_WITH_CA_CERT
        -----END CERTIFICATE-----
      server_cert: |
        -----BEGIN CERTIFICATE-----
        REPLACE_WITH_SERVER_CERT
        -----END CERTIFICATE-----
      server_key: |
        -----BEGIN RSA PRIVATE KEY-----
        REPLACE_WITH_SERVER_KEY
        -----END RSA PRIVATE KEY-----

  your_networking_provider:
    your_policy_client:
      ca_cert: |
        -----BEGIN CERTIFICATE-----
        REPLACE_WITH_CA_CERT
        -----END CERTIFICATE-----
      client_cert: |
        -----BEGIN CERTIFICATE-----
        REPLACE_WITH_CLIENT_CERT
        -----END CERTIFICATE-----
      client_key: |
        -----BEGIN RSA PRIVATE KEY-----
        REPLACE_WITH_CLIENT_KEY
        -----END RSA PRIVATE KEY-----
```

The server requires that connections use the TLS cipher suite
`TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`.  Your client must support this cipher suite.

We provide [a script](../scripts/generate-certs) to generate all required certs & keys.

### Policy Server Internal API Details

`GET /networking/v1/internal/policies`

List all policies optionally filtered to match requested  `policy_group_id`'s

Query Parameters (optional):

- `id`: comma-separated `policy_group_id` values

Response Body:

- `policies`: list of policies
- `policies[].destination`: the destination of the policy
- `policies[].destination.id`: the `policy_group_id` of the destination (currently always an `app_id`)
- `policies[].destination.ports`: the range of `ports` allowed on the destination
- `policies[].destination.ports.start`: the first port in the port range allowed on the destination
- `policies[].destination.ports.end`: the last port of the port range allowed on the destination
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
  https://policy-server.service.cf.internal:4003/networking/v1/internal/policies
```

```json
  {
      "policies": [
        {
            "destination": {
                "id": "eb95ff20-cba8-4edc-8f4a-cf80d0669faf",
                "ports": {
                  "start": 8080,
                  "end": 8090
                },
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
                "ports": {
                  "start": 8080,
                  "end": 8080
                },
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
                "ports": {
                  "start": 8080,
                  "end": 8080
                },
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
                "ports": {
                  "start": 5555,
                  "end": 6666
                },
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
https://policy-server.service.cf.internal:4003/networking/v1/internal/policies?id=5351a742-6704-46df-8de0-1a376adab65c,d5bbc5ed-886a-44e6-945d-67df1013fa16
```

```json
{
    "policies": [
        {
            "destination": {
                "id": "d5bbc5ed-886a-44e6-945d-67df1013fa16",
                "ports": {
                  "start": 5555,
                  "end": 6666
                },
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
                "ports": {
                  "start": 5555,
                  "end": 6666
                },
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
