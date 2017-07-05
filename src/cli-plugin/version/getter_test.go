package version_test

import (
	"cli-plugin/version"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Getter", func() {
	var (
		filename string
		getter   *version.Getter
	)

	BeforeEach(func() {
		filename = writeFile("1.2.1")
		getter = &version.Getter{
			Filename: filename,
		}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(filename)).To(Succeed())
	})

	Describe("Get", func() {
		It("returns the version in the file name", func() {
			versionType, err := getter.Get()
			Expect(err).NotTo(HaveOccurred())

			Expect(versionType.Major).To(Equal(1))
			Expect(versionType.Minor).To(Equal(2))
			Expect(versionType.Build).To(Equal(1))
		})

		Context("when the file does not exist", func() {
			BeforeEach(func() {
				Expect(os.RemoveAll(filename)).To(Succeed())
			})
			It("returns a sensible error", func() {
				_, err := getter.Get()
				Expect(err).To(MatchError(ContainSubstring("file does not exist: ")))
			})
		})

		Context("when the file is not properly-formatted", func() {
			BeforeEach(func() {
				Expect(os.RemoveAll(filename)).To(Succeed())
				filename = writeFile("banana")
				getter = &version.Getter{
					Filename: filename,
				}
			})

			It("returns a sensible error", func() {
				_, err := getter.Get()
				Expect(err).To(MatchError("invalid version: banana"))
			})
		})

		Context("when reading the major number fails", func() {
			BeforeEach(func() {
				Expect(os.RemoveAll(filename)).To(Succeed())
				filename = writeFile("9999999999999999999.0.1")
				getter = &version.Getter{
					Filename: filename,
				}
			})

			It("returns a sensible error", func() {
				_, err := getter.Get()
				Expect(err).To(MatchError("invalid major number: 9999999999999999999"))
			})
		})

		Context("when reading the minor number fails", func() {
			BeforeEach(func() {
				Expect(os.RemoveAll(filename)).To(Succeed())
				filename = writeFile("1.9999999999999999999.0")
				getter = &version.Getter{
					Filename: filename,
				}
			})

			It("returns a sensible error", func() {
				_, err := getter.Get()
				Expect(err).To(MatchError("invalid minor number: 9999999999999999999"))
			})
		})

		Context("when reading the build number fails", func() {
			BeforeEach(func() {
				Expect(os.RemoveAll(filename)).To(Succeed())
				filename = writeFile("1.0.9999999999999999999")
				getter = &version.Getter{
					Filename: filename,
				}
			})

			It("returns a sensible error", func() {
				_, err := getter.Get()
				Expect(err).To(MatchError("invalid build number: 9999999999999999999"))
			})
		})
	})
})

func writeFile(version string) string {
	configFile, err := ioutil.TempFile("", "version.txt")
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(configFile.Name(), []byte(version), os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	return configFile.Name()
}
