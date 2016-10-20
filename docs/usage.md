# Using the CLI

## Install the CF CLI Plugin
1. Get the binary

  - Option 1: Download a precompiled binary of the `network-policy-plugin` for your operating system from our [GitHub Releases](https://github.com/cloudfoundry-incubator/netman-release/releases)

  - Option 2: Build from source

    ```bash
    go build -o /tmp/network-policy-plugin ./src/cli-plugin
    ```

2. Install it

  ```bash
  chmod +x ~/Downloads/network-policy-plugin
  cf install-plugin ~/Downloads/network-policy-plugin
  ```


# Using the API
