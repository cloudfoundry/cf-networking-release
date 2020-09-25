#Adding libraries or packages

This document is intended for contributors who want to extend this codebase.


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
