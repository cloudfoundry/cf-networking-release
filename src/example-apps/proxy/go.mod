module code.cloudfoundry.org/cf-networking-release/src/example-apps/proxy

go 1.13

require (
	example-apps/proxy v0.0.0-00010101000000-000000000000
	github.com/onsi/ginkgo v1.10.3
	github.com/onsi/gomega v1.7.1
)

replace example-apps/proxy => ./
