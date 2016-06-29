#### To use the example app:

Push two differently named instances of the app
```bash
cd ~/workspace/netman-release/src/example-apps/proxy
cf push appA
cf push appB
```

See that they are reachable and what their IPs are
```bash
curl appa.<system-domain>
curl appb.<system-domain>
```

See that proxying from A to B works over both the router and overlay
```bash
curl appa.<system-domain>/proxy/appb.<system-domain>
curl appa.<system-domain>/proxy/<overlay-ip-of-appB>:8080
```

Configure extra ports for the app to listen on
```bash
cf set-env appB USER_PORTS 4444,3333
cf restage
```

See that appB listens on the new ports
```bash
curl appa.<system-domain>/proxy/<overlay-ip-of-appB>:3333
curl appa.<system-domain>/proxy/<overlay-ip-of-appB>:4444
```

