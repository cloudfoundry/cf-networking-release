# Container Networking Performance Suite
Container networking performance tests will be conducted on the following environment: **toque.c2c.cf-app.com**

## Setup
### Number of Diego Cells
0. Git checkout [cf-networking-ci](https://code.cloudfoundry.org/cf-networking-ci) repository under ~/workspace.

    ```
    $ cd ~/workspace && git clone https://github.com/cloudfoundry/cf-networking-ci.git
    ```
0. Set `toque-deploy` `CELL_COUNT` to the number of desired Diego cells in [pipelines/cf-networking.yml](pipelines/cf-networking.yml)
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
              file: cf-networking-release-ci/ci/cf-networking-ci/tasks/generate-deployment-manifests.yml
              params:
                ENVIRONMENT_NAME: toque
                CELL_COUNT: 5
      ```

0.  Update pipeline
    ```
    $ ./reconfigure cf-networking
    ```
*Note: In order to reconfigure a [Concourse](http://concourse.ci) pipeline you may need to first download the `fly` command line tool for [Mac](https://c2c.ci.cf-app.com/api/v1/cli?arch=amd64&platform=darwin), [Windows](https://c2c.ci.cf-app.com/api/v1/cli?arch=amd64&platform=windows) or [Linux](https://c2c.ci.cf-app.com/api/v1/cli?arch=amd64&platform=linux)*

0.  Git commit `CELL_COUNT` changes back to [cf-networking-ci](https://code.cloudfoundry.org/cf-networking-ci)
0.  Deploy `CELL_COUNT` changes using `toque-deploy` stage in [CI](https://c2c.ci.cf-app.com/pipelines/cf-networking/jobs/toque-deploy)
