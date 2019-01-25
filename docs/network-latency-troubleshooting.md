# Latency with Container to Container Networking

You have probably found these docs because you are experiencing latency with container to container networking. 

There are 4 places where the slowness can be coming from: 
- the source app
- dns resolution
- networking
- the destination app

Here are some debugging tools you can try to dig deeper and find the source of the issue. 

If you need more help than this doc, or need help analyzing the results, feel free to reach out to us in the `#container-networking` channel on [Cloud Foundry Slack](http://slack.cloudfoundry.org/).

## Investigate the types of requests 
**Goal:  determine if the backend app's databse queries are the cause of the problem**
#### Some questions to answer: 
- Are all of the requests identical? 
- Do the requests result in touching a database? 
- Do the backend app's database query times show increased times at the time of the latency spike?

## Test your dns resolution speed

**Goal: this should either identify DNS as the issue or eliminate it as the problem**

1. `cf ssh` onto your source app 
2. `dig` the internal route
3. Observe the time the `dig` takes

For example: 
```
$ dig backend.apps.internal
; <<>> DiG 9.11.3-1ubuntu1.3-Ubuntu <<>> backend.apps.internal
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 17122
;; flags: qr aa rd ra; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 0

;; QUESTION SECTION:
;backend.apps.internal.		IN	A

;; ANSWER SECTION:
backend.apps.internal.	0	IN	A	10.255.96.4

;; Query time: 2 msec
;; SERVER: 169.254.0.2#53(169.254.0.2)
;; WHEN: Fri Jan 25 01:58:30 UTC 2019
;; MSG SIZE  rcvd: 76

```
There are 2 important pieces of information here. 
  - `ANSWER 1` or `ANSWER 0`: `1` means that it was able to resolve the route, `0` means it was unable to resolve the route. 
  - `Query time: 2 msec` : This means that the dns resolution took 2 milliseconds. 



 
## Try pushing a go app
**Goal: this should either identify your apps as the issue or eliminate them as the problem**

Push our simple go app and see if there is latency when you curl it. 
[Go app code here.](https://github.com/cloudfoundry/cf-networking-release/tree/develop/src/example-apps/proxy)


1. push the app
  ```
  cf push proxy 
  ```
2. create an internal route
  ```
  cf map-route proxy apps.internal --hostname proxy
  ```
3. create policy from your frontend app to proxy
  ```
  cf add-network-policy FRONTEND-APP --destination-app proxy --protocol tcp --port 8080
  ```
4. get onto the frontend app 
  ```
  cf ssh FRONTEND-APP
  ```
5. time how long curling the same go app takes
  ```
  time curl proxy.apps.internal
  ```

If there is no latency seen the problem likely originates in either your frontend or backend app. If there is latency seen the problem likely originates with the DNS or the networking

## Other things you can try: 

### Try using IPs instead of internal routes
**Goal: another way to eliminte DNS as the source of the problem**
This is not meant a solution to the problem. This should only be used to determine if the slowness is still present when DNS is taken out of the equation. 


1. Look up the overlay IP of the destination app.
2. `cf ssh` onto your source app
3. Curl the destination app by it's overylay IP: `curl DESTINATION-OVERLAY-IP:DESTINATION-APP-PORT`. 
4. Try this for apps that are on the same cell and apps that are on different cells. 

