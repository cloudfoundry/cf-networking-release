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

#### Optional param `returnHeaders`
The Dump requests handler also takes an optional boolean query param
`returnHeaders` that will, when `true`:
- clone the headers sent to the proxy and add them to
  the response headers
- return two additional debug headers: `X-Proxy-Settable-Debug-Header` and
  `X-Proxy-Immutable-Debug-Header`

The value returned by proxy in the `X-Proxy-Settable-Debug-Header` will be
copied from the original request's `X-Proxy-Settable-Debug-Header` header, if
present. As it's name implies, `X-Proxy-Immutable-Debug-Header` cannot be
configured and will *always* return the header with the same value.

```bash
$ curl -v -H 'X-Proxy-Settable-Debug-Header: potato' https://proxy.mydomain.com/dumprequest/?returnHeaders=true
...
> GET /dumprequest/?returnHeaders=true HTTP/1.1
> Host: proxy.mydomain.com
> User-Agent: curl/7.81.0
> Accept: */*
> X-Proxy-Settable-Debug-Header: potato 👈 Setting the debug header, sending to the proxy
>
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< Accept: */*
< B3: 6f098b875f6e433564f0077c97e0a08c-64f0077c97e0a08c
< Content-Length: 572
< Content-Type: text/plain; charset=utf-8
< Date: Tue, 15 Aug 2023 23:36:52 GMT
< User-Agent: curl/7.81.0
< X-B3-Spanid: 64f0077c97e0a08c
< X-B3-Traceid: 6f098b875f6e433564f0077c97e0a08c
< X-Cf-Applicationid: 0b5e54c7-c9ad-4d3a-a0d3-0c351a77c3b2
< X-Cf-Instanceid: 2ace08d0-1160-4159-7e8e-b8ec
< X-Cf-Instanceindex: 0
< X-Forwarded-For: 127.0.0.1
< X-Forwarded-Proto: http
< X-Proxy-Immutable-Debug-Header: default-immutable-value-from-within-proxy-src-code 👈 our immutable header is sent in the response
< X-Proxy-Settable-Debug-Header: potato 👈 proxy is happy to send our debug header back, along with everything else
< X-Request-Start: 1692142612239
< X-Vcap-Request-Id: 6f098b87-5f6e-4335-64f0-077c97e0a08c
<
GET /dumprequest/?returnHeaders=true HTTP/1.1
Host: proxy.mydomain.com
Accept: */*
B3: 6f098b875f6e433564f0077c97e0a08c-64f0077c97e0a08c
User-Agent: curl/7.81.0
X-B3-Spanid: 64f0077c97e0a08c
X-B3-Traceid: 6f098b875f6e433564f0077c97e0a08c
X-Cf-Applicationid: 0b5e54c7-c9ad-4d3a-a0d3-0c351a77c3b2
X-Cf-Instanceid: 2ace08d0-1160-4159-7e8e-b8ec
X-Cf-Instanceindex: 0
X-Forwarded-For: 127.0.0.1
X-Forwarded-Proto: http
X-Proxy-Settable-Debug-Header: potato 👈 the proxy instance received our configued debug header
X-Request-Start: 1692142612239
X-Vcap-Request-Id: 6f098b87-5f6e-4335-64f0-077c97e0a08c
```

## `/echosourceip`

[Echo source IP handler](./handlers/echo_source_ip_handler.go) responds with the
IP address of the client making the request. This can be useful when used in
combination with the `/proxy/` endpoint

```bash
$ curl https://proxy.mydomain.com/
{"ListenAddresses":["127.0.0.1","10.255.208.54"],"Port":8080}

$ curl https://proxy.mydomain.com/proxy/appB.apps.internal/echosourceip
10.255.208.54 # IP address of the proxy app making the request
```

## `/eventuallyfail`

[Eventually Fail handler](./handlers/eventually_fail.go) responds to the first
5 requests with a 200 status code. After the 5th request the endpoint responds
with a 500 status code. You can configure how many times the endpoint should
succeed before failing by setting the `EVENTUALLY_FAIL_AFTER_COUNT` env var.
This endpoint was created to test app healthchecks.

## `/eventuallysucceed`

[Eventually Succeed handler](./handlers/eventually_succeed.go) responds to the first
5 requests with a 500 status code. After the 5th request the endpoint responds
with a 200 status code. You can configure how many times the endpoint should
fail before succeeding by setting the `EVENTUALLY_SUCCEED_AFTER_COUNT` env var.
This endpoint was created to test app healthchecks.

## `/flap`

[Flap handler](./handlers/flap_handler.go) responds to the first 5 requests
with a 200 status code. It responds to the next 5 requests with a 500 status
code. It continues to flap back and forth between 200 and 500 status codes
every 5 requests. You can configure how often the endpoint should flap by
setting the `FLAP_INTERVAL` env var. This endpoint was created to test app
healthchecks.

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

## `/stats`

[Stats handler](./handlers/stats_handler.go) returns latency data for prior
requests to the `/proxy/` endpoint. Can be cleared with the `DELETE` method.

```bash
$ curl https://appa.mydomain.com/stats
{"latency":[]}

$ curl https://appa.mydomain.com/proxy/appB.apps.internal:8080
$ curl https://appa.mydomain.com/proxy/appB.apps.internal:8080
$ curl https://appa.mydomain.com/proxy/appB.apps.internal:8080
$ curl https://appa.mydomain.com/proxy/appB.apps.internal:8080

$ curl https://appa.mydomain.com/stats
{"latency":[0.006587316,0.007722343,0.005693,0.005313626]}

$ curl https://appa.mydomain.com/stats -X DELETE
$ curl https://appa.mydomain.com/stats
{"latency":[]}
```

## `/timed_dig/${domain}`

[Timed dig handler](./handlers/timed_dig_handler.go) like `/dig/`, but with
timing information in the response.

```bash
$ curl https://proxy.mydomain.com/dig/example.com
{"lookup_time_ms":2,"ips":["93.184.216.34","\u003cnil\u003e"]}
```

## `/upload`
[Upload handler](./handlers/upload_handler.go) returns the number of bytes
received in the request body.

```bash
$ curl https://proxy.mydomain.com/download/1024 > foo
$ curl https://proxy.mydomain.com/upload --data-binary @foo
1024 bytes received and read
```
