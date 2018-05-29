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
$ cf curl /networking/v0/external/policies
{"total_policies":2,"policies":[{"source":{...}]}
```

### Option 2: curl
When using curl the token must be explicitly provided in the `Authorization` header.

Example
```sh
$ export TOKEN=`cf oauth-token` # as CF admin
$ curl http://api.bosh-lite.com/networking/v0/external/policies -H "Authorization: $TOKEN"
{"total_policies":2,"policies":[{"source":{...}]}
```

## API Documentation

| Method | Path | Arguments | Request Body | Description|
| :----- | :--- | :-------- | :----------- | :----------- |
| GET | /networking/v0/external/policies | [see below](#get-networkingv0externalpolicies) | - | List Policies |
| POST | /networking/v0/external/policies | - | [see below](#post-networkingv0externalpolicies)| Create Policies |
| POST | /networking/v0/external/policies/delete | - | [see below](#post-networkingv0externalpoliciesdelete)| Delete Policies |
| GET | /networking/v0/external/tags | - | - | List all tag and `id` mappings |

Notes:
A unique tag is assigned to a policy_group_id when policies are created.

### GET /networking/v0/external/policies
#### Arguments:

[optionally] `id`: comma-separated app id values\
[optionally] `source_id`: comma-separated source app id values\
[optionally] `dest_id`: comma-separated destination app id values

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
        "port": 1234
      }
    },
    {
      "source": {
        "id": "308e7ef1-63f1-4a6c-978c-2e527cbb1c36"
      },
      "destination": {
        "id": "308e7ef1-63f1-4a6c-978c-2e527cbb1c36",
        "protocol": "tcp",
        "port": 1234
      }
    }
  ]
}
```

### POST /networking/v0/external/policies

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
        "port": 1234
      }
    },
    {
      "source": {
        "id": "308e7ef1-63f1-4a6c-978c-2e527cbb1c36"
      },
      "destination": {
        "id": "308e7ef1-63f1-4a6c-978c-2e527cbb1c36",
        "protocol": "tcp",
        "port": 1234
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
| destination.port | Y | The destination port (1 - 65535)

#### Response Status Codes:
- 200 (successful)
- 400 (invalid request)

### POST /networking/v0/external/policies/delete

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
        "port": 1234
      }
    },
    {
      "source": {
        "id": "308e7ef1-63f1-4a6c-978c-2e527cbb1c36"
      },
      "destination": {
        "id": "308e7ef1-63f1-4a6c-978c-2e527cbb1c36",
        "protocol": "tcp",
        "port": 1234
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
| destination.port | Y | The destination port (1 - 65535)

#### Response Status Codes:
- 200 (successful)
- 400 (invalid request)

### GET /networking/v0/external/tags

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
