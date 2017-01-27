## Contributing
We welcome contributions from the community.  Here are guidelines for development.

### Running low-level tests

```bash
~/workspace/cf-networking-release/scripts/docker-test
```

### Running the full acceptance test on bosh-lite
WARNING: This test is taxing and has an aggressive timeout.
It may fail on a laptop or other underpowered bosh-lite.

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
