# Tips for Large Deployments with CF Networking and Silk Release 

Some users have larger deployments than we regularly test with. We have heard of large deployments with 500-1000 diego cells. 
These deployments have specific considerations that smaller deployments don't need to worry about. 

Please submit a PR or create an issue if you have come across other large deployment considerations.

## Problem 1: Silk Daemon uses too much CPU
### Symptoms
The silk daemon begins using too much CPU on the cells. This causes the app health checks to fail, which causes the apps to evacuate the cell.

### Reason
The silk daemon is deployed on every cell. It is in charge of getting the IP leases for every other cell from the silk controller. The 
silk daemon calls out to the silk controller every 5 seconds (by default) to get updated lease information. Every time it gets new information 
the silk daemon does some linux system calls to set up the networking. This can take a long time (relatively) and get expensive when there are a lot of 
cells with new leases. This causes the silk daemons to use a lot of CPU.

### Solution
Change the property `lease_poll_interval_seconds` on the silk-daemon job to be greater than 5 seconds. This will cause the silk-daemon to 
poll the silk-controller less frequently and thus make linux system calls less frequently. However, increasing this property means that when a cell gets a new lease 
(this happens when a cell is rolled, recreated, or for whatever reason it doesn't renew it's lease properly) it will take longer for the other cells to know how to 
route container-to-container traffic to it.

## Problem 2: ARP Cache on diego-cell not large enough 
### Symptoms
Silk daemon fails to converge leases. Errors in the silk-daemon logs might look like this: 
```json
{  
   "timestamp": "TIME",
   "source": "cfnetworking.silk-daemon",
   "message": "cfnetworking.silk-daemon.poll-cycle",
   "log_level": 2,
   "data": {  
      "error":"converge leases: del neigh with ip/hwaddr 10.255.21.2 : no such file or directory"
   }
}
```

### Reason
ARP cache on the diego cell is not large enough to handle the number of entries the silk-daemon is trying to write.

### Solution
Increase the ARP cache size on the diego cells. 
