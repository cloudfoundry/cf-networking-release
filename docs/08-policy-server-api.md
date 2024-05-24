---
title: Policy Server
expires_at: never
tags: [cf-networking-release]
---

<!-- vim-markdown-toc GFM -->

* [External API](#external-api)
  * [API Authorization](#api-authorization)
    * [Option 1: cf curl](#option-1-cf-curl)
    * [Option 2: curl](#option-2-curl)
  * [API Documentation](#api-documentation)
    * [GET /networking/v1/external/policies](#get-networkingv1externalpolicies)
      * [Arguments:](#arguments)
      * [Response Body:](#response-body)
    * [POST /networking/v1/external/policies](#post-networkingv1externalpolicies)
      * [Request Body:](#request-body)
    * [POST /networking/v1/external/policies/delete](#post-networkingv1externalpoliciesdelete)
      * [Request Body:](#request-body-1)
      * [Response Status Codes:](#response-status-codes)
    * [GET /networking/v1/external/tags](#get-networkingv1externaltags)
      * [Response Body:](#response-body-1)
* [Internal API](#internal-api)
  * [Policy Server Internal API Details](#policy-server-internal-api-details)
    * [Example Put Tags Request and Response](#example-put-tags-request-and-response)
      * [Create a new tag](#create-a-new-tag)
    * [Example Get Policy Request and Response](#example-get-policy-request-and-response)
      * [Get all policies](#get-all-policies)
      * [Get Filtered Policies](#get-filtered-policies)
      * [Get Security Groups](#get-security-groups)

<!-- vim-markdown-toc -->
# External API

The policy server API is used for creating, deleting and listing policies and
tags.

## API Authorization
In order to communicate with the policy server API, a UAA oauth token with valid
`network.admin` or `network.write` scope is required.  The CF admin by default
has `network.admin` scope, other users will need to have the proper scope
granted by an admin.

Space developers with the `network.write` scope can configure policies for
applications in spaces for which they have the SpaceDeveloper role.

### Option 1: cf curl
Use the `cf curl` command as admin

Example

```bash
$ cf curl /networking/v1/external/policies
{"total_policies":2,"policies":[{"source":{...}]}
```

### Option 2: curl
When using curl the token must be explicitly provided in the `Authorization` header.

Example
```bash
$ export TOKEN=`cf oauth-token` # as CF admin
$ curl http://api.bosh-lite.com/networking/v1/external/policies -H "Authorization: $TOKEN"
{"total_policies":2,"policies":[{"source":{...}]}
```

## API Documentation

The current API is v1.

| Method | Path | Arguments | Request Body | Description|
| :----- | :--- | :-------- | :----------- | :----------- |
| GET | /networking/v1/external/policies | [see below](#get-networkingv1externalpolicies) | - | List Policies |
| POST | /networking/v1/external/policies | - | [see below](#post-networkingv1externalpolicies)| Create Policies |
| POST | /networking/v1/external/policies/delete | - | [see below](#post-networkingv1externalpoliciesdelete)| Delete Policies |
| GET | /networking/v1/external/tags | - | - | List all tag and `id` mappings |

Notes:
- A policy_group_id is a generic way to identify a policy, but currently it is
  also the same as the app guid
- A unique tag is assigned to a policy_group_id when policies are created.

### GET /networking/v1/external/policies
#### Arguments:

[optionally] `id`: comma-separated policy_group_id values\
[optionally] `source_id`: comma-separated source policy_group_id values\
[optionally] `dest_id`: comma-separated destination policy_group_id values

Will return only the policies which include the given policy_group_id either as source id or destination id.

#### Response Body:

```json
{
  "total_policies": 2,
  "policies": [
    {
      "source": {
        "id": "1081ceac-f5c4-47a8-95e8-88e1e302efb5"
      },
      "destination": {
        "id": "38f08df0-19df-4439-b4e9-61096d4301ea",
        "protocol": "tcp",
        "ports": {
          "start": 1234,
          "end": 1235
        }
      }
    },
    {
      "source": {
        "id": "308e7ef1-63f1-4a6c-978c-2e527cbb1c36"
      },
      "destination": {
        "id": "308e7ef1-63f1-4a6c-978c-2e527cbb1c36",
        "protocol": "tcp",
        "ports": {
          "start": 1234,
          "end": 1235
        }
      }
    }
  ]
}
```

### POST /networking/v1/external/policies

#### Request Body:

```json
{
  "policies": [
    {
      "source": {
        "id": "1081ceac-f5c4-47a8-95e8-88e1e302efb5"
      },
      "destination": {
        "id": "38f08df0-19df-4439-b4e9-61096d4301ea",
        "protocol": "tcp",
        "ports": {
          "start": 1234,
          "end": 1235
        }
      }
    },
    {
      "source": {
        "id": "308e7ef1-63f1-4a6c-978c-2e527cbb1c36"
      },
      "destination": {
        "id": "308e7ef1-63f1-4a6c-978c-2e527cbb1c36",
        "protocol": "tcp",
        "ports": {
          "start": 1234,
          "end": 1235
        }
      }
    }
  ]
}
```

| Field | Required? | Description |
| :---- | :-------: | :------ |
| policies.source.id | Y | The source `policy_group_id`
| policies.destination.id | Y | The destination `policy_group_id`
| policies.destination.protocol | Y | The protocol (tcp or udp)
| policies.destination.ports | Y | The destination port range
| policies.destination.ports.start | Y | The destination start port (1 - 65535)
| policies.destination.ports.end | Y | The destination end port (1 - 65535)

### POST /networking/v1/external/policies/delete

#### Request Body:

```json
{
  "policies": [
    {
      "source": {
        "id": "1081ceac-f5c4-47a8-95e8-88e1e302efb5"
      },
      "destination": {
        "id": "38f08df0-19df-4439-b4e9-61096d4301ea",
        "protocol": "tcp",
        "ports": {
          "start": 1234,
          "end": 1235
        }
      }
    },
    {
      "source": {
        "id": "308e7ef1-63f1-4a6c-978c-2e527cbb1c36"
      },
      "destination": {
        "id": "308e7ef1-63f1-4a6c-978c-2e527cbb1c36",
        "protocol": "tcp",
        "ports": {
          "start": 1234,
          "end": 1235
        }
      }
    }
  ]
}
```

| Field | Required? | Description |
| :---- | :-------: | :------ |
| policies.source.id | Y | The source `policy_group_id`
| policies.destination.id | Y | The destination `policy_group_id`
| policies.destination.protocol | Y | The protocol (tcp or udp)
| policies.destination.ports | Y | The destination port range
| policies.destination.ports.start | Y | The destination start port (1 - 65535)
| policies.destination.ports.end | Y | The destination end port (1 - 65535)

#### Response Status Codes:
- 200 (successful)
- 400 (invalid request)
- 406 (unsupported API version)

### GET /networking/v1/external/tags

#### Response Body:

```json
{
  "tags": [
    {
      "id": "1081ceac-f5c4-47a8-95e8-88e1e302efb5",
      "tag": "0001"
    },
    {
      "id": "308e7ef1-63f1-4a6c-978c-2e527cbb1c36",
      "tag": "0002"
    },
    {
      "id": "38f08df0-19df-4439-b4e9-61096d4301ea",
      "tag": "0003"
    }
  ]
}
```

# Internal API

If you are replacing the built-in "VXLAN Policy Agent" with your own Policy
Enforcement implementation, you can use the Policy Server's internal API to
retrieve policy information.

There is a single endpoint to retrieve policies:

`GET https://policy-server.service.cf.internal:4003/networking/v1/internal/policies`

Additionally, you can use the `id` query parameter to filter the response to
include only policies with a source or destination that match any of the
comma-separated `group_policy_id`'s that are included.

## Policy Server Internal API Details

`PUT /networking/v1/internal/tags`

Create a new tag for a given `type` and `id`
Noop and returns existing tag if present

If a request is made for an existing `id` with a new `type`, the request will
fail. `id` is a unique constraint.

Json Parameters (required):
- `id`: a unique identifier for the resource or group of resources; e.g. `INGRESS_ROUTER`
- `type`: the type of the group being requested for; e.g. `router`

Example Request Body:
```json
{
  "id": "INGRESS_ROUTER",
  "type": "router"
}
```

Response Body:

- `ID`: the id supplied in the request
- `Type`: the type supplied in the request
- `Tag`: the tag assigned to the group

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

`GET /networking/v1/internal/security_groups`

List security groups that are bound to spaces defined by `space_guids` parameter and global security groups.

Query Parameters (optional):

- `space_guids`: comma-separated values of space guids
- `limit`: the number of security groups to return
- `from`: the id of the security group to start the returned values from

Response Body:

- `next`: the id of the security group that goes next after the security groups
  in the response. 0 indicates that there are no more security groups to follow
- `security_groups`: list of security groups
- `security_groups[].guid`: the guid of the security group
- `security_groups[].name`: the name of the security group
- `security_groups[].rules`: the JSON array of [security group
  rules](https://docs.cloudfoundry.org/concepts/asg.html#creating-groups)
- `security_groups[].staging_default`: whether the security group is global for
  staging application containers
- `security_groups[].running_default`: whether the security group is global for
  running application containers
- `security_groups[].staging_space_guids`: comma-separated list of staging space
  guids the security group is bound to
- `security_groups[].running_space_guids`: comma-separated list of running space
  guids the security group is bound to

### Example Put Tags Request and Response

#### Create a new tag

```bash
curl \
  --cacert ca.crt \
  --cert client.crt \
  --key client.key \
  https://policy-server.service.cf.internal:4003/networking/v1/internal/tags \
  -X PUT \
  -d '{ "id": "router", "type": "fakeType" }'
```

```json
{
  "id": "router",
  "type": "fakeType",
  "tag": "0004"
}
```

### Example Get Policy Request and Response

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
          "end": 8090,
          "start": 8080
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
          "end": 8080,
          "start": 8080
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
          "end": 8080,
          "start": 8080
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
          "end": 6666,
          "start": 5555
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

#### Get Filtered Policies

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

#### Get Security Groups

Return security groups that are bound to provided space guids as well as global
security groups.

```bash
curl -s \
--cacert certs/ca.crt \
--cert certs/client.crt \
--key certs/client.key \
https://policy-server.service.cf.internal:4003/networking/v1/internal/security_groups?space_guids=5351a742-6704-46df-8de0-1a376adab65c,d5bbc5ed-886a-44e6-945d-67df1013fa16
```

```json
{
  "next": 0,
  "security_groups": [
    {
      "guid": "b4669e65-e196-4c4d-8504-913e69d525bb",
      "name": "public_networks",
      "rules": "[{\"protocol\":\"all\",\"destination\":\"0.0.0.0-9.255.255.255\",\"ports\":\"\",\"type\":0,\"code\":0,\"description\":\"\",\"log\":false},{\"protocol\":\"all\",\"destination\":\"11.0.0.0-169.253.255.255\",\"ports\":\"\",\"type\":0,\"code\":0,\"description\":\"\",\"log\":false},{\"protocol\":\"all\",\"destination\":\"169.255.0.0-172.15.255.255\",\"ports\":\"\",\"type\":0,\"code\":0,\"description\":\"\",\"log\":false},{\"protocol\":\"all\",\"destination\":\"172.32.0.0-192.167.255.255\",\"ports\":\"\",\"type\":0,\"code\":0,\"description\":\"\",\"log\":false},{\"protocol\":\"all\",\"destination\":\"192.169.0.0-255.255.255.255\",\"ports\":\"\",\"type\":0,\"code\":0,\"description\":\"\",\"log\":false}]",
      "staging_default": true,
      "running_default": true,
      "staging_space_guids": [],
      "running_space_guids": []
    },
    {
      "guid": "cb2d7b6a-0d91-4f63-a69c-4ebcb5c683a3",
      "name": "dns",
      "rules": "[{\"protocol\":\"tcp\",\"destination\":\"0.0.0.0/0\",\"ports\":\"53\",\"type\":0,\"code\":0,\"description\":\"\",\"log\":false},{\"protocol\":\"udp\",\"destination\":\"0.0.0.0/0\",\"ports\":\"53\",\"type\":0,\"code\":0,\"description\":\"\",\"log\":false}]",
      "staging_default": true,
      "running_default": true,
      "staging_space_guids": [],
      "running_space_guids": []
    },
    {
      "guid": "c1654f71-ead9-4a16-bc7e-10835cfea7f1",
      "name": "security-group-1",
      "rules": "[{\"protocol\":\"icmp\",\"destination\":\"0.0.0.0/0\",\"ports\":\"\",\"type\":0,\"code\":0,\"description\":\"\",\"log\":false},{\"protocol\":\"tcp\",\"destination\":\"10.0.11.0/24\",\"ports\":\"80,443\",\"type\":0,\"code\":0,\"description\":\"Allow http and https traffic to ZoneA\",\"log\":true}]",
      "staging_default": false,
      "running_default": false,
      "staging_space_guids": [],
      "running_space_guids": [
        "5351a742-6704-46df-8de0-1a376adab65c"
      ]
    },
    {
      "guid": "e22dcb62-e310-4aae-b560-9692cc809277",
      "name": "security-group-2",
      "rules": "[{\"protocol\":\"icmp\",\"destination\":\"0.0.0.0/0\",\"ports\":\"\",\"type\":0,\"code\":0,\"description\":\"\",\"log\":false},{\"protocol\":\"tcp\",\"destination\":\"10.0.11.0/24\",\"ports\":\"80,443\",\"type\":0,\"code\":0,\"description\":\"Allow http and https traffic to ZoneA\",\"log\":true}]",
      "staging_default": false,
      "running_default": false,
      "staging_space_guids": [
        "d5bbc5ed-886a-44e6-945d-67df1013fa16"
      ],
      "running_space_guids": []
    }
  ]
}

```
