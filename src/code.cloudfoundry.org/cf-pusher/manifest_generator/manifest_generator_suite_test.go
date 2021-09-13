package manifest_generator_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestManifestGenerator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ManifestGenerator Suite")
}
