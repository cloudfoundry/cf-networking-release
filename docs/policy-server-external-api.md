# Policy Server External API

## Purpose:

The policy server API is used for creating, deleting and listing policies and tags.

## API Authorization
In order to communicate with the policy server API, a UAA oauth token with valid `network.admin` or `network.write` scope is required.
The CF admin by default has `network.admin` scope, other users will need to have the proper scope granted by an admin.

Space developers with the `network.write` scope can configure policies for applications in spaces for which they have the SpaceDeveloper role.

### Option 1: cf curl
Use the `cf curl` command as admin

Example
```sh
$ cf curl /networking/v1/external/policies
{"total_policies":2,"policies":[{"source":{...}]}
```

### Option 2: curl
When using curl the token must be explicitly provided in the `Authorization` header.

Example
```sh
$ export TOKEN=`cf oauth-token` # as CF admin
$ curl http://api.bosh-lite.com/networking/v1/external/policies -H "Authorization: $TOKEN"
{"total_policies":2,"policies":[{"source":{...}]}
```

## API Documentation

The current API is v1.

Earlier versions:

- [API v0](API_v0.md)

| Method | Path | Arguments | Request Body | Description|
| :----- | :--- | :-------- | :----------- | :----------- |
| GET | /networking/v1/external/policies | [see below](#get-networkingv1externalpolicies) | - | List Policies |
| POST | /networking/v1/external/policies | - | [see below](#post-networkingv1externalpolicies)| Create Policies |
| POST | /networking/v1/external/policies/delete | - | [see below](#post-networkingv1externalpoliciesdelete)| Delete Policies |
| GET | /networking/v1/external/tags | - | - | List all tag and `id` mappings |

Notes:
A unique tag is assigned to a policy_group_id when policies are created.

### GET /networking/v1/external/policies
#### Arguments:

[optionally] `id`: comma-separated policy_group_id values

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
| source.id | Y | The source `policy_group_id`
| destination.id | Y | The destination `policy_group_id`
| destination.protocol | Y | The protocol (tcp or udp)
| destination.ports | Y | The destination port range
| destination.ports.start | Y | The destination start port (1 - 65535)
| destination.ports.end | Y | The destination end port (1 - 65535)

#### Response Status Codes:
- 200 (successful)
- 400 (invalid request)
- 406 (unsupported API version)

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
| source.id | Y | The source `policy_group_id`
| destination.id | Y | The destination `policy_group_id`
| destination.protocol | Y | The protocol (tcp or udp)
| destination.ports | Y | The destination port range
| destination.ports.start | Y | The destination start port (1 - 65535)
| destination.ports.end | Y | The destination end port (1 - 65535)

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
