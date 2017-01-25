# tick ❤
Simple app that registers itself with an [a8registry](https://github.com/amalgam8/amalgam8/tree/master/registry) on a regular interval.

## Prerequisites
The following instructions for this example assume the following:
- Go 1.6+
- [cf-networking-release](http://github.com/cloudfoundry-incubator/cf-networking-release)
  - cloned under `~/workspace/cf-networking-release`
- Ensure GOPATH is set to cf-networking-release:
  - export GOPATH=~/workspace/cf-networking-release
- [jq](https://stedolan.github.io/jq/download/)
- Deploying to [bosh-lite](https://github.com/cloudfoundry/bosh-lite)
  - Cloud Foundry org and space created and targetted

## Setup
- Build and Deploy the [service registry](https://github.com/amalgam8/amalgam8/tree/master/registry)
```bash
cd ~/workspace/cf-networking-release/src/github.com/amalgam8/amalgam8
GOOS=linux GOARCH=amd64 go build -o a8registry cmd/registry/main.go
cf push registry -c './a8registry' -b binary_buildpack -d bosh-lite.com
```

## Example
Push 3 instances of `tick` app
```bash
cd ~/workspace/cf-networking-release/src/example-apps/tick
cf push tick -i 3 -m 32M --no-start
cf set-env tick REGISTRY_BASE_URL "http://registry.bosh-lite.com"
cf start tick
```

**Verify the # of app instances registered in service registry**
```bash
$ curl -s registry.bosh-lite.com/api/v1/instances | jq '.instances | length'
  3
```

**See details of app instances registered in service registry**
```bash
$ curl -s registry.bosh-lite.com/api/v1/instances | jq .
{
  "instances": [
    {
      "id": "29bfde7bcd321ee9",
      "service_name": "tick",
      "endpoint": {
        "type": "tcp",
        "value": "10.255.6.9:8080"
      },
      "ttl": 10,
      "status": "UP",
      "last_heartbeat": "2016-09-14T13:30:48.726709455Z"
    },
    {
      "id": "b36fdd8c20018133",
      "service_name": "tick",
      "endpoint": {
        "type": "tcp",
        "value": "10.255.6.10:8080"
      },
      "ttl": 10,
      "status": "UP",
      "last_heartbeat": "2016-09-14T13:30:48.107560504Z"
    },
    {
      "id": "bd1a694d32055f34",
      "service_name": "tick",
      "endpoint": {
        "type": "tcp",
        "value": "10.255.27.13:8080"
      },
      "ttl": 10,
      "status": "UP",
      "last_heartbeat": "2016-09-14T13:30:48.268383996Z"
    }
  ]
}
```
