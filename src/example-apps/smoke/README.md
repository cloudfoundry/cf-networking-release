#### Smoke

This app performs a smoke test on itself. This is used in [Diego CI](`https://github.com/cloudfoundry/diego-ci`).

#### To use the example app:

Push a proxy app named `smokeproxy`.

Push a smoke app with the `no-start` flag, set the environment variable `PROXY_APP_URL` to the smoke app and start the app:

```bash
cd ~/workspace/cf-networking-release/src/example-apps/smoke
cf push smoke --no-start
cf set-env smoke PROXY_APP_URL http://smokeproxy.<sys-domain>.com
cf start smoke
```

Add a network policy between `smokeproxy` and `smoke`, then you're ready to test the app!
```
cf add-network-policy smokeproxy smoke
curl smoke.<sys-domain>.com/selfproxy/
```
