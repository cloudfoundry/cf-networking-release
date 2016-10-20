package manifest_generator_test

import (
	"cf-pusher/manifest_generator"
	"cf-pusher/models"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
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
