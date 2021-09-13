#Adding libraries or packages

This document is intended for contributors who want to extend this codebase.


### Referencing a new library from existing BOSH package

1. Add any new libraries into the submodules from the root of the repo
  ```bash
  cd $RELEASE_DIR/src/code.cloudfoundry.org
  go get github.com/foo/bar #add dependency a go package
  ```

### Sync BOSH package


1. Run the sync script:
  ```bash
  ./scripts/sync-package-specs
  ```
