# Policy Server Internal API

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
