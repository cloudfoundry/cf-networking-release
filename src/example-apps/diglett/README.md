#### To use the example app:

This app continuously runs the command `dig` on a destination.

Push a diglett app with the `no-start` flag, set the environment variables
`DIGLETT_DESTINATION` to the destination and `DIGLETT_FREQUENCY_MS` to the
frequency of the dig command, and start the app:

```bash
cd ~/workspace/cf-app-sd-release/src/example-apps/diglett
cf push diglett --no-start
cf set-env diglett DIGLETT_DESTINATION google.com
cf set-env diglett DIGLETT_FREQUENCY_MS 100
cf start diglett
```
