# Spring Boot Eureka Demo

Simple demo of service registration and discovery using Eureka.

Initial work based on this guide:

https://spring.io/guides/gs/service-registration-and-discovery/

## Assumptions

- `maven` installed (`brew install maven`)
- CF deployed with [cf-networking](https://github.com/cloudfoundry-incubator/cf-networking)
  (The examples assume a bosh-lite deployment)
- `cf` cli installed with [`network-policy-plugin`](https://github.com/cloudfoundry-incubator/cf-networking/blob/develop/docs/usage.md)

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
`frontend` to reach it, policies must be added.

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

### Remove access

```
cf remove-access zuul-proxy backend --protocol tcp --port 8080
cf remove-access frontend backend --protocol tcp --port 8080
```

## More Details

These properties are set to have the `backend` register with it's IP
address in the file `backend/src/main/resources/application.properties`:

```
eureka.client.preferIpAddress=true
eureka.instance.hostname=${CF_INSTANCE_INTERNAL_IP}
eureka.instance.nonSecurePort=${PORT}
```

This causes the backend instance to report its own address as the internal container-network address and port
(not the external, NAT'ed address that the router uses to reach it).


The `backend` reaches the registry at the public address `registry.bosh-lite.com`.
The `zuul-proxy` is also configured to look up
services registered in eureka at this address. Edit this address if deploying
to a CF on a different domain.


