# Information for operators

### MTU
Operators should not need to do any special configuration for MTUs.  The CNI plugins
should automatically detect the host MTU and set the container MTU appropriately, 
accounting for any overhead.

However, operators should understand that:
 - All Diego cells should be on the same network, and should have the same MTU
 - A change the Diego cell MTU will likely require the VMs to be recreated in
   order for the container network to function properly.
