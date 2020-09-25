## Service Discovery Metrics

Below are a list of metrics for service discovery components. To deploy a firehose nozzle to see the metrics, upload the
[datadog-firehose-nozzle-release](http://bosh.io/releases/github.com/DataDog/datadog-firehose-nozzle-release) and follow
the instructions [here](https://github.com/DataDog/datadog-firehose-nozzle-release) to deploy.

Metric Name | Description
------------ | -------------
`bosh_dns_adapter.GetIPsRequestTime` | duration of get ip request in milliseconds
`bosh_dns_adapter.GetIPsRequestCount` | number of get ip requests
`bosh_dns_adapter.DNSRequstFailures` | number of failed requests to the Service Discovery Controller
`bosh_dns_adapter.uptime` | process uptime, emitted on 10 second interval
`service_discovery_controller.RegistrationRequestTime` | duration of registration request in milliseconds
`service_discovery_controller.RegistrationRequestCount` | number of registration requests
`service_discovery_controller.addressTableLookupTime` | duration of looking up address table in milliseconds
`service_discovery_controller.uptime` | process uptime, emitted on 10 second interval
`service_discovery_controller.dnsRequest` | count of successful dnsRequests, emitted on a 10 second interval
`service_discovery_controller.registerMessagesReceived` | count of route register messages received via NATS from route emitter
`service_discovery_controller.maxRouteMessageTimePerInterval` | maximum time taken from BBS to SDC, only on new app creation

