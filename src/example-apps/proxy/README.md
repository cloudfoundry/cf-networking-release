# proxy

The `proxy` app was initially designed to enable testing inter-application connectivity on
using Cloud Foundry's container networking features:

Push two differently named instances of the app
```bash
cd ~/workspace/cf-networking-release/src/example-apps/proxy
cf push appA
cf push appB
```

See that they are reachable and what their IPs are
```bash
curl appa.<system-domain>
curl appb.<system-domain>
```

See that proxying from A to B works using its public route
```bash
curl appa.<system-domain>/proxy/appb.<system-domain>
```

See that proxying from A to B works using its overlay IP address with after adding a network policy
```bash
cf add-network-policy appA appB
curl appa.<system-domain>/proxy/<overlay-ip-of-appB>:8080
```

See that proxying from A to B works using an internal route with after adding a network policy
```bash
cf map-route appB apps.internal --hostname appB
curl appa.<system-domain>/proxy/appB.apps.internal:8080
```

It has since grown to include some additional endpoints useful for testing
networking features.

# Endpoints

## `/`

[Info handler](./handlers/info_handler.go) returns the overly IP address and
ports that the application container is listening on.

Example:
```bash
$ curl https://proxy.mydomain.com/
{"ListenAddresses":["127.0.0.1","10.255.208.82"],"Port":8080}
```

## `/dig/${domain}`

[Dig handler](./handlers/dig_handler.go) uses `net.LookupIP` to return the IP
addresses for the domain requested as a path variable to the endpoint

```bash
$ curl https://proxy.mydomain.com/dig/example.com
["93.184.216.34","\u003cnil\u003e"]
```

## `/digudp/${domain}`

[Dig UDP handler](./handlers/dig_udp_handler.go) shells out to `dig` with
`+notcp` to look up the IP addresses for the domain requested as a path variable
to the endpoint using UDP.

```bash
$ curl https://proxy.mydomain.com/dig/example.com
["93.184.216.34"]
```

## `/download/${numbytes}`

[Download handler](./handlers/download_handler.go) responds with a randomly
generated body of length in bytes equal to `numbytes`

```bash
$ curl https://proxy.mydomain.com/download/1024 > foo && wc -c foo
    1024 foo
```

## `/dumprequest/`

[Dump request handler](./handlers/dump_request_handler.go) dumps the request
headers received and responds with them in the response body.

```bash
$ curl https://proxy.mydomain.com/dumprequest -H "X-MyHeader: foo"
GET /dumprequest/ HTTP/1.1
Host: proxy.mydomain.com
Accept: */*
B3: d154a35fda473f91-d154a35fda473f91
User-Agent: curl/7.64.1
Via: 1.1 google
X-B3-Spanid: d154a35fda473f91
X-B3-Traceid: d154a35fda473f91
X-Cf-Applicationid: cf11aec9-2d64-4af4-99e6-40374ded9146
X-Cf-Instanceid: c8c2d611-7517-4150-7a13-90ee
X-Cf-Instanceindex: 0
X-Cloud-Trace-Context: f6d833aeb67e61ea7526e690d4b34df7/14789025218083271155
X-Forwarded-For: 76.175.68.86, 34.96.66.201, 35.191.8.81
X-Forwarded-Proto: https
X-Myheader: foo
X-Request-Start: 1600296938338
X-Vcap-Request-Id: 2249e92a-fbc0-42e3-4d33-e341cd3969c8
```

## `/echosourceip/`

[Echo source IP handler](./handlers/echo_source_ip_handler.go) responds with the
IP address of the client making the request. This can be useful when used in
combination with the `/proxy/` endpoint

```bash
$ curl https://proxy.mydomain.com/
{"ListenAddresses":["127.0.0.1","10.255.208.54"],"Port":8080}

$ curl https://proxy.mydomain.com/proxy/appB.apps.internal/echosourceip/
10.255.208.54 # IP address of the proxy app making the request
```

## `/ping/${destination}`

[Ping handler](./handlers/ping_handler.go) shells out to `ping` to ping the
destination address and responds with success or failure.

```bash
$ curl https://proxy.mydomain.com/ping/example.com
Ping succeeded to destination: example.com

$ curl https://proxy.mydomain.com/ping/schmexample.com
Ping failed to destination: schmexample.com: exit status 2
```

## `/proxy/${destination}`

[Proxy handler](./handlers/proxy_handler.go) makes an HTTP request to
`${destination}` and returns the response. Useful for testing network
connectivity between applications:

```bash
$ cf add-network-policy appA appB
$ cf map-route appB apps.internal --hostname appB
$ curl https://appa.mydomain.com/proxy/appB.apps.internal:8080
{"ListenAddresses":["127.0.0.1","10.255.208.82"],"Port":8080}
```

## `/stats/`

[Stats handler](./handlers/stats_handler.go) returns latency data for prior
requests to the `/proxy/` endpoint. Can be cleared with the `DELETE` method.

```bash
$ curl https://appa.mydomain.com/stats/
{"latency":[]}

$ curl https://appa.mydomain.com/proxy/appB.apps.internal:8080
$ curl https://appa.mydomain.com/proxy/appB.apps.internal:8080
$ curl https://appa.mydomain.com/proxy/appB.apps.internal:8080
$ curl https://appa.mydomain.com/proxy/appB.apps.internal:8080

$ curl https://appa.mydomain.com/stats/
{"latency":[0.006587316,0.007722343,0.005693,0.005313626]}

$ curl https://appa.mydomain.com/stats/ -X DELETE
$ curl https://appa.mydomain.com/stats/
{"latency":[]}
```

## `/timed_dig/${domain}`

[Timed dig handler](./handlers/timed_dig_handler.go) like `/dig/`, but with
timing information in the response.

```bash
$ curl https://proxy.mydomain.com/dig/example.com
{"lookup_time_ms":2,"ips":["93.184.216.34","\u003cnil\u003e"]}
```

## `/upload/`
[Upload handler](./handlers/upload_handler.go) returns the number of bytes
received in the request body.

```bash
$ curl https://proxy.mydomain.com/download/1024 > foo
$ curl https://proxy.mydomain.com/upload/ --data-binary @foo
1024 bytes received and read
```
