#### Spammer

This app spams requests on a destination. This is used in [Outbound Connection Limit Test](https://github.com/cloudfoundry/cf-networking-release/blob/3684d1f70b3b35eed727c19d910776955dbc2276/src/code.cloudfoundry.org/test/acceptance/outbound_conn_limit_test.go).

#### To use the example app:

Push a proxy app named `spamtarget`.

Push a spammer app with the `no-start` flag, set the environment variables
`PROXY_URL` to the destination and start the app:

```bash
cd ~/workspace/cf-networking-release/src/example-apps/spammer
cf push spammer --no-start
cf set-env spammer PROXY_URL http://spamtarget.<sys-domain>.com
cf start spammer
```

Then start the spam...  a lot!
```
curl spammer.<sys-domain>.com/spam/<spam-count>
```
