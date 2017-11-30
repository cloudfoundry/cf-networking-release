# cf-networking-ci
Public config for the Container Networking team's CI

Keep credentials and other private config in our [private repo](https://github.com/cloudfoundry/cf-networking-deployments)

The main source code repo is [cf-networking-release](https://code.cloudfoundry.org/cf-networking-release).

View our team's [CI dashboard](http://dashboard.c2c.cf-app.com).

[Metrics dashboard](https://p.datadoghq.com/sb/f3af7f8e2-baf5212773?tv_mode=true).

## to update a pipeline
```bash
./reconfigure $PIPELINE_NAME
```
where `$PIPELINE_NAME` might be `cf-networking`, `images`, etc.
