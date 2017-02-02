# CLI
Network policies can be managed using the CF Networking CLI plugin.

## Installation
1. Get the cf cli plugin binary

  - Option 1: Download a precompiled binary of the `network-policy-plugin` for your operating system from our [GitHub Releases](https://github.com/cloudfoundry-incubator/cf-networking-release/releases)

  - Option 2: Build from source

    ```bash
    go build -o /tmp/network-policy-plugin ./src/cli-plugin
    ```

2. Install it

  ```bash
  chmod +x ~/Downloads/network-policy-plugin
  cf install-plugin ~/Downloads/network-policy-plugin
  ```

## Usage

### Allow Policy:

Allow direct network traffic from one app to another

```
$ cf allow-access -h
NAME:
   allow-access - Allow direct network traffic from one app to another

USAGE:
   cf allow-access SOURCE_APP DESTINATION_APP --protocol <tcp|udp> --port <1-65535>

OPTIONS:
   --port           Port to connect to destination app with. (required)
   --protocol       Protocol to connect apps with. (required)
```

**Example:**
```sh
$ cf allow-access frontend backend --protocol tcp --port 8080
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
   cf remove-access SOURCE_APP DESTINATION_APP --protocol <tcp|udp> --port <1-65535>

OPTIONS:
   --port           Port to connect to destination app with. (required)
   --protocol       Protocol to connect apps with. (required)
```

**Example:**
```sh
$ cf remove-access frontend backend --protocol tcp --port 8080
Denying traffic from frontend to backend as admin...
OK
```
