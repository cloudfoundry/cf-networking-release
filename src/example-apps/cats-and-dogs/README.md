# Cats and Dogs

A sample app to demonstrate communication (HTTP and UDP) between a frontend and a backend application over the container network.

We're assuming that you've [deployed to BOSH lite](../../../docs/bosh-lite.md).
If you've [deployed to AWS](../../../docs/iaas.md#deploy-to-aws) or another environment,
substitute `bosh-lite.com` below with the domain name of your installation.

To configure policies you must have the CF Networking
[CLI plugin](https://github.com/cloudfoundry-incubator/cf-networking-release/blob/develop/docs/CLI.md) installed.


## Frontend
The frontend serves a form at `http://frontend.bosh-lite.com/`.

The frontend allows you to test out container network communication via two methods:

- connect to the backend via HTTP
- connect to the backend via UDP

In either case, the response from the backend to the frontend will be rendered as a web page.


### Deploying
```
cd cf-networking-release/src/example-apps/cats-and-dogs/frontend
cf api api.bosh-lite.com --skip-ssl-validation
cf auth admin admin
cf push frontend
```


## Backend
The backend displays its container network IP if you visit `http://backend.bosh-lite.com/`.

The backend serves pictures of cats on the TCP ports specified in the environment variable `CATS_PORTS`,
and responds to simple text messages on the UDP ports specified in the environment variable `UDP_PORTS`.


### Deploying
```
cd cf-networking-release/src/example-apps/cats-and-dogs/backend
cf push backend --no-start
cf set-env backend CATS_PORTS "7007,7008"
cf set-env backend UDP_PORTS "9003,9004"
cf start backend
```


## Usage

After both frontend and backend apps have been deployed, you can visit `http://backend.bosh-lite.com/`
in a browser. You should see something like:

```
My overlay IP is: 10.255.76.2

I'm serving cats on TCP ports 7007,7008

I'm also serving a UDP echo server on UDP ports 9003,9004
```

If you were to visit `http://frontend.bosh-lite.com`, you should see something like:

```
Frontend Sample App

HTTP Test
Backend HTTP URL: [....] [ Submit ]

UDP Test
Backend UDP Server Address: [....]
Message: [....] [ Submit ]
```


### Usage with HTTP

In `Backend HTTP URL` enter the backend's overlay IP and a cats port (10.255.76.2:7007).
Hit submit.

You will see an error message. This is because the two apps have not been
configured to allow connections from the frontend to the backend.

Now allow access:

```
cf allow-access frontend backend --port 7007 --protocol tcp
```

Now if you try again from the frontend:

```
[PICTURE OF CAT]
Hello from the backend, port: 7007
```


### Usage with UDP

In `Backend UDP Server Address` enter the backend's overlay IP and UDP port
(10.255.76.2:9003) and a message. Hit submit.

You will see an error message. This is because the two apps have not been
configured to allow connections from the frontend to the backend.

Now allow access:

```
cf allow-access frontend backend --port 9003 --protocol udp
```

Now if you try again from the frontend:

```
You sent the message: hello world

Backend UDP server replied: HELLO WORLD
```
