## Contributing
We welcome contributions from the community.  Here are guidelines for development.

### Running unit, integration and template tests

```bash
~/workspace/cf-networking-release/scripts/docker-test
~/workspace/cf-networking-release/scripts/template-tests
```

### Running the full acceptance test on bosh-lite
#### Setting up

Run the [`scripts/deploy-to-bosh-lite`](scripts/deploy-to-bosh-lite) script.

To deploy, [cf-networking-release](https://github.com/cloudfoundry/cf-networking-release), [bosh-deployment](https://github.com/cloudfoundry/bosh-deployment), and [cf-deployment](https://github.com/cloudfoundry/cf-deployment) repos are required.

#### Running acceptance
```bash
cd src/test/acceptance
./run-locally.sh
```

### Referencing a new library from existing BOSH package
1. Add any new libraries into the submodule from the root of the repo

  ```bash
  cd $GOPATH
  git submodule add https://github.com/foo/bar src/github.com/foo/bar
  ./scripts/sync-package-specs
  ```

### Adding a new BOSH package
1. Add any new libraries into the submodules from the root of the repo
  ```bash
  cd $GOPATH
  git submodule add https://github.com/foo/bar src/github.com/foo/bar
  ```

2. Update the package sync script:
  ```bash
  vim $GOPATH/scripts/sync-package-specs
  ```
  Find or create the `sync_package` line for `baz`

3. Run the sync script:
  ```bash
  ./scripts/sync-package-specs
  ```

### When using bosh-lite, not finding iptable logging inside kern.log
The linux kernel prevents iptable log targets from working inside a container.
See [commit introducing the change](https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/commit/?id=69b34fb996b2eee3970548cf6eb516d3ecb5eeed)
