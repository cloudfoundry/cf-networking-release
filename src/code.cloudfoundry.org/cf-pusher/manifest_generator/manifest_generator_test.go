package manifest_generator_test

import (
	"io/ioutil"

	"code.cloudfoundry.org/cf-pusher/manifest_generator"
	"code.cloudfoundry.org/cf-pusher/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ManifestGenerator", func() {
	var (
		manifestGenerator *manifest_generator.ManifestGenerator
		object            models.Manifest
	)
	BeforeEach(func() {
		manifestGenerator = &manifest_generator.ManifestGenerator{}
		object = models.Manifest{
			Applications: []models.Application{{
				Name:   "test-name",
				Memory: "test-memory",
				Env: models.TickEnvironment{
					GoPackageName: "test-package/testme",
				},
			}},
		}
	})
	It("it should marshal the object into a file", func() {
		file, err := manifestGenerator.Generate(object)
		Expect(err).NotTo(HaveOccurred())

		Expect(file).To(BeAnExistingFile())
		contents, err := ioutil.ReadFile(file)
		Expect(err).NotTo(HaveOccurred())
		Expect(contents).To(MatchYAML(`
applications:
  - name: test-name
    memory: test-memory
    env:
      GOPACKAGENAME: test-package/testme
`))
	})
})
