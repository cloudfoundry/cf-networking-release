# CF Networking CI

Public config for the Container Networking team's CI

Keep credentials and other private config in our [private repo](https://github.com/cloudfoundry/networking-oss-deployments)

The main source code repo is [cf-networking-release](https://code.cloudfoundry.org/cf-networking-release).

View our team's [CI dashboard](https://networking.ci.cf-app.com/teams/ga/pipelines/cf-networking).

## to update a pipeline

```bash
./reconfigure $PIPELINE_NAME
```

where `$PIPELINE_NAME` might be `cf-networking`, `images`, etc.
