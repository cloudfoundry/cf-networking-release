## Service Discovery Architecture

### Architecture Diagram
![](architecture-diagram.png)

Routes are emitted from the Route Emitter. Internal routes are emitted from the
Route Emitter as well, on a separate topic.

The NATS message queue cluster that handles routes for the gorouter also handles
internal routes.

The Service Discovery Controller (SDC) subscribes to route updates from NATS on
the internal routes topic. The SDC is highly available. The SDC has no
persistence, it is an in memory store of internal domain names to IPs. The SDC
warms (populates routes) before entering service.

Each Diego Cell has a BOSH DNS and a BOSH-DNS Adapter. App containers are
configured to use the BOSH DNS server on their Deigo cell as their DNS server.
The BOSH-DNS Adapter configures BOSH DNS to route queries for internal domains
to itself. When a request for an internal domain hits BOSH DNS it looks at the
domain name. If it's internal it directs the request to the BOSH-DNS Adapter.
BOSH DNS communicates to the BOSH DNS Adapter via http (following the [Google
DNS over
HTTPS](https://developers.google.com/speed/public-dns/docs/dns-over-https)
schema).

The BOSH DNS adapter in turn makes a request to the SDC. This HTTP connection is
secured using mTLS. Responses from the SDC contain all the IPs of all the app
containers associated with the requested route. Responses from the BOSH DNS
adapter contain all the IPs returned from the SDC, shuffled. BOSH DNS in turn
returns the full set of IPs originally from the SDC. Clients typically use the
first IP in the DNS response, the shuffling provides a crude form of load
balancing.