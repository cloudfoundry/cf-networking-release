# CLI
Network policies can be managed using the CF Networking CLI plugin. Policies are currently configured between applications.
Any tasks that are created will receive the same policies that the app it is associated with has.

## Installation

There are 3 ways to install the plugin.  The easiest way to get started is using option 1.

#### Option 1: Using the Community Plugin Repository

1. Ensure you have a recent version of the CF CLI:

   ```
   cf version
   ```

   Should show version `6.28` or higher.
   If not, update your CF CLI by following the [CLI installation instructions](http://docs.cloudfoundry.org/cf-cli/install-go-cli.html).

2. Install the plugin from the [Cloud Foundry Community Plugins Repository](https://plugins.cloudfoundry.org/):

   ```
   cf install-plugin -r CF-Community network-policy
   ```

#### Option 2: Using a binary release from GitHub

1. Download a precompiled binary of the `network-policy-plugin` for your
   operating system from our [GitHub Releases](https://github.com/cloudfoundry-incubator/cf-networking-release/releases)

2. Install the binary

    ```bash
    chmod +x ~/Downloads/network-policy-plugin
    cf install-plugin ~/Downloads/network-policy-plugin
    ```

#### Option 3: Building from source

  From the root of this repository, run

  ```bash
  direnv allow
  go build -o /tmp/network-policy-plugin ./src/cli-plugin
  cf install-plugin /tmp/network-policy-plugin
  ```

## Usage

### Allow Policy:

Allow direct network traffic from one app to another

```
$ cf allow-access -h
NAME:
   allow-access - Allow direct network traffic from one app to another

USAGE:
   cf allow-access SOURCE_APP DESTINATION_APP --protocol <tcp|udp> --port <1-65535>[-<1-65535>]

OPTIONS:
   --port           Port(s) to connect to destination app with. (required)
   --protocol       Protocol to connect apps with. (required)
```

**Example:**
```sh
$ cf allow-access frontend backend --protocol tcp --port 8080-8090
Allowing traffic from frontend to backend as admin...
OK
```

### List Policies:

List policy for direct network traffic from one app to another

```
$ cf list-access
NAME:
   list-access - List policy for direct network traffic from one app to another

USAGE:
   cf list-access [--app appName]

OPTIONS:
   --app       Application to filter results by. (optional)
```

**Example:**
```sh
$ cf list-access
Listing policies as admin...
OK

Source		Destination	Protocol	Port
frontend	backend		tcp		8080
```

### Remove Policy:

Remove direct network traffic from one app to another

```
$ cf remove-access -h
NAME:
   remove-access - Remove policy and deny direct network traffic from one app to another

USAGE:
   cf remove-access SOURCE_APP DESTINATION_APP --protocol <tcp|udp> --port <1-65535>[-<1-65535>]

OPTIONS:
   --port           Port(s) to connect to destination app with. (required)
   --protocol       Protocol to connect apps with. (required)
```

**Example:**
```sh
$ cf remove-access frontend backend --protocol tcp --port 8080-8090
Denying traffic from frontend to backend as admin...
OK
```
