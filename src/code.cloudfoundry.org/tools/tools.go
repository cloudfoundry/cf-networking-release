//go:build tools
// +build tools

package tools

import (
	_ "code.cloudfoundry.org/locket/cmd/locket"
	_ "github.com/onsi/ginkgo/ginkgo"
)

// This file imports packages that are used when running go generate, or used
// during the development process but not otherwise depended on by built code.
