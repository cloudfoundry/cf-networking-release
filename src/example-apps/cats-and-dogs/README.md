# Cats and Dogs

A sample app to demonstrate communication between a frontend and a backend application.

We're assuming that you've [deployed to BOSH lite](../../../docs/bosh-lite.md).  If you've [deployed to AWS](../../../docs/aws.md) or
another environment, substitute `bosh-lite.com` below with the domain name of your installation.

## Frontend
The frontend serves a form at `http://frontend.bosh-lite.com/` that allows you to enter a backend URL whose contents will be fetched and rendered.

### Deploying
```
cd cf-networking/src/example-apps/cats-and-dogs/frontend
cf api api.bosh-lite.com --skip-ssl-validation
cf auth admin admin
cf push frontend
```

## Backend
The backend displays its container network IP if you visit `http://backend.bosh-lite.com/` and it serves pictures of cats on the ports specified in the environment variable `CATS_PORTS`

### Deploying
```
cd cf-networking/src/example-apps/cats-and-dogs/backend
cf push backend --no-start
cf set-env backend CATS_PORTS "5678,9876"
cf start backend
```

## Usage

After both apps have been deployed, you can visit `http://backend.bosh-lite.com/` in a browser. You should see something like:

```
My overlay IP is: 10.255.76.2

I'm serving cats on ports 5678,9876
```

If you were to visit `http://frontend.bosh-lite.com` you should see something like:

```
Frontend

Backend URL: [_____] [ Submit ]
```

Enter the backend's overlay IP and port (10.255.76.2:9876) and hit submit. You will see an error message. This is because the two apps have not been configured to allow connections from the frontend to the backend.

To configure policies you must have the CF Networking [CLI plugin](https://github.com/cloudfoundry-incubator/cf-networking/blob/develop/docs/CLI.md) installed. Now allow access:

```
cf allow-access frontend backend --port 9876 --protocol tcp
```

Now if you were to try entering the backend app's overlay IP and port again in the frontend you will see:

```
[PICTURE OF CAT]
Hello from the backend, port: 9876
```
