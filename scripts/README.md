# Scripts

This is the README for our scripts. To learn more about `routing-release`, go to the main [README](../README.md).

| Name | Purpose | Notes |
| --- | --- | --- |
| commit-with-submodule-log | lightweight script for submodule bumps, allows for commits that don't finish a story | depends on submodule-log |
| create-container | convenience script for creating a garden container | |
| deploy-to-bosh-lite | creates, uploads, and deploys local release to a local bosh-lite | |
| docker-shell | starts a docker image based on a database, use `db=` to set `mysql` or `mysql-5.6` or `postgres` | |
| docker-shell-with-started-db | same as docker-shell, but starts the database for you | |
| docker-test | uses docker-shell to run unit-and-integration-tests | |
| generate-copilot-proto | generates the copilot protobuf files for bosh-dns-adapter | |
| list-dependencies | generates a list of component dependencies | |
| start-db-in-docker | boots the database in a docker | used by docker-shell-with-started-db |
| submodule-log | prints the cached submodule log and if you provide story id(s) will add finishes tag(s) | |
| sync-package-specs | calls sync-package-specs_linux-only in a docker | |
| sync-package-specs_linux-only | updates the packages spec for each component | |
| template-tests | runs the template spec tests for the release | |
| test-acceptance | runs the acceptance tests against a provided bosh environment | |
| test-sd-acceptance | runs the service discovery acceptance tests against a provided bosh environment | |
| test-sd-acceptance-local | runs the service discovery acceptance tests against a local bosh-lite | |
| test-sd-performance | runs the service discovery performance tests against a provided bosh environment | |
| unit-and-integration-tests | runs unit and integration tests for networking components | |
| update | updates all submodules | |
