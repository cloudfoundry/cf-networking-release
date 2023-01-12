# Example Apps Overview

This doc provides an overview of the example apps provided and the uses cases they provide.

## Use Cases
* [general network debugging](#use-case-you-want-an-app-to-do-network-debugging)
* [use container-to-container networking](#use-case-you-want-to-try-out-container-to-container-networking)
* [use container-to-container networking with service discovery](#use-case-you-want-to-try-out-container-to-container-networking-with-service-discovery)
* [use Eureka for service discovery](#use-case-you-want-to-try-out-eureka-for-service-discovery)
* [use an a8registry](#use-case-you-want-to-try-out-eureka-for-service-discovery)

<hr>

## Use case: you want an app to do network debugging
**App:** [Proxy](https://github.com/cloudfoundry/cf-networking-release/tree/develop/src/example-apps/proxy)

**Description**: Proxy has endpoints for using dig and ping, for showing stats, uploading and downloading an arbitrary number of bytes, and more. It is a good app for general debugging purposes. 


## Use case: you want to try out container-to-container networking
**App:** [Cats and Dogs](https://github.com/cloudfoundry-attic/cf-networking-examples/blob/master/docs/c2c-no-service-discovery.md)

**Description:** This example demonstrates container-to-container networking via HTTP and UDP between a frontend and backend app.


## Use case: you want to try out container-to-container networking with service discovery
**App:** [Cats and Dogs with Service Discovery](https://github.com/cloudfoundry-attic/cf-networking-examples/blob/master/docs/c2c-with-service-discovery.md)

**Description:** This example demonstrates container-to-container networking via HTTP and UDP between a frontend and backend app with service discovery.


## Use case: you want to use an a8registry
**App:** [tick](https://github.com/cloudfoundry/cf-networking-release/tree/develop/src/example-apps/tick)

**Description:** This is a simple app that registers itself with an a8registry on a regular interval.
