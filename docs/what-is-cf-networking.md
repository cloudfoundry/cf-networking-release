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