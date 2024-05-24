---
title: What is CF Networking 
expires_at: never
tags: [silk-release]
---

<!-- vim-markdown-toc GFM -->

* [What is CF Networking?](#what-is-cf-networking)
  * [What Does CF Networking Provide?](#what-does-cf-networking-provide)
  * [Motivation for Container to Container (c2c) Networking](#motivation-for-container-to-container-c2c-networking)
  * [Motivation for Policies for C2C Networking](#motivation-for-policies-for-c2c-networking)
  * [Motivation for Service Discovery for C2C Networking](#motivation-for-service-discovery-for-c2c-networking)
* [Architecture](#architecture)
    * [Core components](#core-components)
    * [Batteries included, but swappable](#batteries-included-but-swappable)
    * [Plugin layer cake](#plugin-layer-cake)

<!-- vim-markdown-toc -->
# What is CF Networking?

## What Does CF Networking Provide?

This release provides three main functionalities: 
* **container to container (c2c) networking** - is the ability for apps in one CF foundation to talk directly to other apps.
* **policies for c2c networking** - is the ability to limit which apps can use c2c to talk to other apps.
* **service discovery for c2c networking** - is the ability to use routes for c2c communication.


## Motivation for Container to Container (c2c) Networking
Before this release, when one app on Cloud Foundry wanted to talk to another app on Cloud Foundry the traffic would have to exit the foundation and re-enter again through the load balancer. Not only did this add unnecesary latency, but it could also be a security risk. With microservices, there is often no need to expose backend apps to the internet and doing so adds an unnecesary attack vector.

With c2c functionality apps can send traffic directly to other apps in Cloud Foundry.

```
Without CF Networking

+-------------------------------+
|                               |
|                               v                 With CF Networking
|                        +------+------+
|                        |Load Balancer|          +---------------------------+
|                        +------+------+          |Diego Cell                 |
|                               |                 |                           |
|                               v                 |                           |
|                          +----+---+             |  +--------+    +-------+  |
|                          |Gorouter|             |  |Frontend|    |Backend|  |
|                          +----+---+             |  |  App   +--->+  App  |  |
|                               |                 |  +--------+    +-------+  |
|         +---------------------------+           |                           |
|         |Diego Cell           |     |           +---------------------------+
|         |                     |     |
|         |                     v     |
|         |  +--------+    +----+--+  |
|         |  |Frontend|    |Backend|  |
+------------+  App   |    |  App  |  |
          |  +--------+    +-------+  |
          |                           |
          +---------------------------+

```

## Motivation for Policies for C2C Networking
Policies give admins and space developers the ability to explicitly state who is allowed to access apps via c2c. Backend apps often have access to sensative user information and policies provide more security for this information.

## Motivation for Service Discovery for C2C Networking
C2c networking works by sending traffic to an app instances container IP. These container IPs change everytime a new app conainer is made. For example, apps get new instance IPs when the Diego Cells roll during a CF deployment. 

Before service discovery apps would have to figure out the container IPs for all of the app instances they wanted to talk to. Often this was done through a 3rd party service discovery service like Eurika or Amalgam8.

# Architecture

CF Networking provides policy-driven container networking for Cloud Foundry.

CF Networking has several components.  Some are "core" to the Cloud Foundry
platform, others are "swappable" by operators who wish to use a 3rd party
network system instead.  For more information on integrating a 3rd-party
networking solution, [see here](11-3rd-party.md).

![](diagram.png)

### Core components

- Policy Server, a central management node, exposes a JSON REST API used by the CLI plugin
- [Garden External Networker](../src/code.cloudfoundry.org/garden-external-networker), a [Garden-runC](https://github.com/cloudfoundry/garden-runc-release) add-on deployed to every Diego cell
  - Invokes an operator-configured [CNI](https://github.com/containernetworking/cni) Plugin to set up the network for each app instance (container)
  - Forwards ports to support incoming connections from the CF [HTTP Router](https://docs.cloudfoundry.org/concepts/http-routing.html),
    [TCP Router](https://docs.cloudfoundry.org/adminguide/enabling-tcp-routing.html) and [Diego SSH Proxy](https://docs.cloudfoundry.org/concepts/diego/ssh-conceptual.html).

### Batteries included, but swappable
On every Diego cell
- [Silk](https://github.com/cloudfoundry/silk), provides IP address management and network connectivity to app instances (containers)
  - Uses a [VXLAN overlay](data_plane.png) for sending traffic between cells
  - Every CF app instance gets a unique IP on a shared, flat L3 network
- VXLAN Policy Agent enforces network policy for network traffic between applications
  - Discovers desired network policies from the [Policy Server's Internal API](11-3rd-party.md#policy-server-internal-api)
  - Updates IPTables rules on Diego cell to allow whitelisted ingress traffic
  - Egress traffic is tagged with a unique identifier per source application, using the [VXLAN GBP header](https://tools.ietf.org/html/draft-smith-vxlan-group-policy-02#section-2.1)
- Traffic destined for container IPs travels in the overlay network. This traffic is subject to container to container network policies.
- Traffic destined for the Internet or any other non container IPs travels in the underlay network. This traffic is subject to application security groups and dynamic ASG network policies.

| Multi Diego Cell |
:-------------------------:
| ![](data_plane.png) |

Single Diego Cell | ASG
:-------------------------:|:-------------------------:
![](data_plane_one_cell.png)  |  ![](data_plane_asg.png)

### Plugin layer cake
Here is a summary of the network-related actions that occur when a new container is created.

![](plugin-layer-cake.png)
