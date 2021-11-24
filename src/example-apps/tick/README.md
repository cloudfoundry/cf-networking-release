# tick ❤
Simple app that registers itself with an
[a8registry](https://github.com/amalgam8/amalgam8/tree/master/registry) on a
regular interval.

## Prerequisites
The following instructions for this example assume the following:
- [This git repo](http://github.com/cloudfoundry/cf-networking-release) cloned somewhere
- [jq](https://stedolan.github.io/jq/download/)
- A Cloud Foundry deployed.  Below we assume
  [bosh-lite](https://github.com/cloudfoundry/bosh-lite), but you can substitute
  your CF domain instead.

## Example
Starting from this directory....

Push an instance of the [service registry](https://github.com/amalgam8/amalgam8/tree/master/registry)
```bash
cd ../registry
cf push registry
cd ../tick
```

Push 3 instances of `tick` app
```bash
cf push tick --no-start
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
