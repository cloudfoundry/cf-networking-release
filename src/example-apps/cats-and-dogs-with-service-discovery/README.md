# Cats and Dogs

A sample app to demonstrate communication (HTTP and UDP) between a frontend and a backend application over the container network.

This app also demonstrates how to use service discovery with container networking.
To see this, you must also deploy with [service discovery](https://github.com/cloudfoundry/cf-app-sd-release) enabled.

We're assuming that you've [deployed to BOSH lite](https://github.com/cloudfoundry/cf-deployment).
If you've [deployed to AWS](https://github.com/cloudfoundry/cf-deployment) or another environment,
substitute `bosh-lite.com` below with the domain name of your installation.

To configure policies and create internal routes you use the [CF CLI](https://github.com/cloudfoundry/cli).


## Frontend
The frontend serves a form at `http://frontend.bosh-lite.com/`.

The frontend allows you to test out container network communication via two methods:

- connect to the backend via HTTP
- connect to the backend via UDP

In either case, the response from the backend to the frontend will be rendered as a web page.


### Deploying
```
cd cf-networking-release/src/example-apps/cats-and-dogs-with-service-discovery/frontend
cf api api.bosh-lite.com --skip-ssl-validation
cf auth admin admin
cf create-org cats-and-dogs
cf target -o cats-and-dogs
cf create-space cats
cf target -o cats-and-dogs -s cats
cf push frontend
```


## Backends
The backend will be pushed with no external route and therefore should not be accessible via the public internet.

Backend-A serves a picture of a typing cat on the TCP ports specified in the environment variable `CATS_PORTS`,
and responds to simple text messages on the UDP ports specified in the environment variable `UDP_PORTS`.

Backend-B serves a picture of a party cat on the TCP ports specified in the environment variable `CATS_PORTS`,
and responds to simple text messages on the UDP ports specified in the environment variable `UDP_PORTS`.

We will give both backends internal hostnames that will map to the app's container ips and can be used to connect
via container-to-container networking. An internal hostname is configured via the CF CLI `map-route` command, with
the domain provided set to the reserved internal domain of `apps.internal`.

We will also create a third internal route that maps to both apps. When queried, the route should return both apps
as possible destinations.

### Deploying
Backend-A
```
cd cf-networking-release/src/example-apps/cats-and-dogs-with-service-discovery/backend-a
cf push backend-a --no-start --no-route
cf map-route backend-a apps.internal --hostname backend-a
cf set-env backend-a CATS_PORTS "7007,7008"
cf set-env backend-a UDP_PORTS "9003,9004"
cf start backend-a
```

Backend-B
```
cd cf-networking-release/src/example-apps/cats-and-dogs-with-service-discovery/backend-b
cf push backend-b --no-start --no-route
cf map-route backend-b apps.internal --hostname backend-b
cf set-env backend-b CATS_PORTS "7007,7008"
cf set-env backend-b UDP_PORTS "9003,9004"
cf start backend-b
```

Creating an internal route shared across both backends
```
cf create-route cats apps.internal --hostname backend
cf map-route backend-a apps.internal --hostname backend
cf map-route backend-b apps.internal --hostname backend
```

## Usage

After both frontend and backend apps have been deployed, you can visit `http://frontend.bosh-lite.com/`
in a browser. You should see something like:

```
Frontend Sample App

HTTP Test
Backend HTTP URL: [....] [ Submit ]

UDP Test
Backend UDP Server Address: [....]
Message: [....] [ Submit ]
```


### Usage with HTTP

In `Backend HTTP URL` enter backend-a's internal hostname and a cats port (`backend-a.apps.internal:7007`).
Hit submit.

You will see an error message saying the connection is refused. This is because the two apps have not been
configured to allow connections from the frontend to the backend.

If you see an error message saying `no such host`, service discovery is incorrectly configured.

Now allow access:

```
cf add-network-policy frontend --destination-app backend-a --port 7007 --protocol tcp
```

Now if you try again from the frontend:

```
[GIF OF CAT]
Hello from the backend, port: 7007
```

Doing the same thing with backend-b should result in a different cat (namely, a party cat) being shown.

When policy is configured for the frontend to reach both backend-a and backend-b on the same port, entering
the shared internal hostname and port (`backend.apps.internal:7007`) will show
```
[GIF OF CAT]
Hello from the backend, port: 7007
```

Trying this multiple times should result in seeing both cat gifs returned (as both apps are routed via that hostname).

### Usage with UDP

In `Backend UDP Server Address` enter the backend's internal hostname and UDP port
(`backend-a.apps.internal:9003`) and a message. Hit submit.

You will see an error message. This is because the two apps have not been
configured to allow connections from the frontend to the backend.

Now allow access:

```
cf add-network-policy frontend --destination-app backend-a --port 9003 --protocol udp
```

Now if you try again from the frontend:

```
You sent the message: hello world

Backend UDP server replied: HELLO WORLD
```
