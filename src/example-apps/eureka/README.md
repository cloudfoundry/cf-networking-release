# Spring Boot Eureka Demo

Simple demo of service registration and discovery using Eureka.

Initial work based on this guide:

https://spring.io/guides/gs/service-registration-and-discovery/

## Assumptions

- `maven` installed (`brew install maven`)
- CF deployment with [netman](https://github.com/cloudfoundry-incubator/netman-release)
  (The examples assume a bosh-lite deployment)
- `cf` cli [`network-policy-plugin`](https://github.com/cloudfoundry-incubator/netman-release/blob/develop/docs/usage.md) is installed

## Build the applications

There are 4 applications:

- `registry`: The Eureka Service Registry
- `backend`: A simple web service that registers with the `registry`
- `frontend`: A web app that locates the `backend` via the `registry`
- `zuul-proxy`: A web proxy that forwards requests to services in the `registry`

```
./build
```

## Push the applications

The `manifest.yml` includes all 4 applications.
```
cf push
```

## Container Networking

The `backend` is configured with the `no-route:true` property to disable
accessing it via the `go-router`. In order for the `zuul-proxy` or the
`frontend` to reach it, policies must be added to allow access.

### Allow access from frontend to backend

```
cf allow-access frontend backend --protocol tcp --port 8080
```

http://frontend.bosh-lite.com/

should have the same response:

```
{
  springApplicationName: "backend",
  serverPort: "8080"
}
```


### Allow access from zuul-proxy to backend

```
cf allow-access zuul-proxy backend --protocol tcp --port 8080
```

http://zuul-proxy.bosh-lite.com/backend/whoami

should respond with:

```
{
  springApplicationName: "backend",
  serverPort: "8080"
}
```

In addition to the `/whoami`, the following actuator endpoints are also available:

- /backend/instances
- /backend/health
- /backend/metrics
- /backend/mappings

### Deny access

```
cf deny-access zuul-proxy backend --protocol tcp --port 8080
cf deny-access frontend backend --protocol tcp --port 8080
```

## More Details

These properties are set to have the `euerka-client` register with it's IP
address:

- `spring.cloud.inetutils.preferredNetworks`
- `eureka.client.preferIpAddress=true`

They are both set in the `application.properties` file, but the former is
overridden via an environment variable in the manifest to choose an IP address
on the container overlay network:
`SPRING_CLOUD_INETUTILS_PREFERREDNETWORKS: 10.255`

The `backend` registers with the `registry` via it's public address,
`registry.bosh-lite.com`.  The `zuul-proxy` is also configured to look up
services registered in eureka at this address. Edit this address if deploying
to a CF on a different domain.


