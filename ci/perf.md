# Container Networking Performance Suite

Container networking performance tests will be conducted on the following
environment: **toque.c2c.cf-app.com**

## Setup
### Number of Diego Cells

1. Make sure you have cf-networking-release checked out in your workspace.

    ```bash
    $ cd ~/workspace && git clone https://github.com/cloudfoundry/cf-networking-release.git
    ```

1. Set `toque-deploy` `CELL_COUNT` to the number of desired Diego cells in
   [ci/pipelines/cf-networking.yml](pipelines/cf-networking.yml)

     ```yaml
      - name: toque-deploy
        serial_groups: [toque]
        plan:
          - aggregate:
            - get: cf-networking-release-ci
              trigger: false
               # omit for brevity
          - aggregate:
               # omit for brevity
            - task: generate-toque-manifests
              file: cf-networking-release-ci/ci/tasks/generate-deployment-manifests.yml
              params:
                ENVIRONMENT_NAME: toque
                CELL_COUNT: 5
      ```

1.  Update pipeline

    ```bash
    $ ./reconfigure cf-networking
    ```

> Note: In order to reconfigure a [Concourse](http://concourse.ci) pipeline you
> may need to first download the `fly` command line tool for
> [Mac](https://networking.ci.cf-app.com/api/v1/cli?arch=amd64&platform=darwin),
> [Windows](https://networking.ci.cf-app.com/api/v1/cli?arch=amd64&platform=windows) or
> [Linux](https://networking.ci.cf-app.com/api/v1/cli?arch=amd64&platform=linux)

1.  Git commit `CELL_COUNT` changes back to this repo.
1.  Deploy `CELL_COUNT` changes using `toque-deploy` stage in
    [CI](https://networking.ci.cf-app.com/teams/ga/pipelines/cf-networking/jobs/toque-deploy)
