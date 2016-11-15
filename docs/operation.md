# Information for operators

### MTU
Operators should not need to do any special configuration for MTUs.  The CNI plugins
should automatically detect the host MTU and set the container MTU appropriately, 
accounting for any overhead.

However, operators should understand that:
 - All Diego cells should be on the same network, and should have the same MTU
 - A change the Diego cell MTU will likely require the VMs to be recreated in
   order for the container network to function properly.

### Mutual TLS
The policy server exposes its internal API over mutual TLS.  We provide [a script](../scripts/generate-certs)
to generate these certificates for you.  If you want to generate them yourself,
ensure that the certificates support the cipher suite `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`.
The Policy Server will reject connections using any other cipher suite.
