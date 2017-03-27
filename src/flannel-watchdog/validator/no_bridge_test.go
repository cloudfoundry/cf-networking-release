package validator_test

import (
	"encoding/json"
	"flannel-watchdog/validator"
	"fmt"
	"io/ioutil"
	"lib/datastore"
	"os"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("NoBridge", func() {
	Describe("Validate", func() {
		var (
			noBridge         *validator.NoBridge
			logger           lager.Logger
			metadataFileName string
		)

		BeforeEach(func() {
			data := map[string]datastore.Container{
				"container-1": datastore.Container{
					Handle: "some-handle",
					IP:     "10.244.40.1",
				},
			}

			metadata, err := json.Marshal(data)
			Expect(err).NotTo(HaveOccurred())

			metadataFile, err := ioutil.TempFile("", "")
			Expect(err).NotTo(HaveOccurred())
			metadataFileName = metadataFile.Name()
			err = ioutil.WriteFile(metadataFileName, metadata, os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			logger = lagertest.NewTestLogger("test")
			noBridge = &validator.NoBridge{
				Logger:           logger,
				MetadataFileName: metadataFileName,
			}
		})

		Context("when the container ips fall within the subnet env range", func() {
			It("returns successfully", func() {
				err := noBridge.Validate("10.244.40.0/12")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the container ips are outside subnet env range", func() {
			It("returns an error", func() {
				err := noBridge.Validate("10.10.40.10/24")
				Expect(err).To(MatchError(`This cell must be restarted (run "bosh restart <job>").  Flannel is out of sync with current containers.`))
			})
		})

		Context("when the metadata file does not exist", func() {
			BeforeEach(func() {
				err := os.Remove(metadataFileName)
				Expect(err).NotTo(HaveOccurred())
			})

			It("logs and return nil", func() {
				err := noBridge.Validate("10.10.40.10/24")
				Expect(err).NotTo(HaveOccurred())

				Expect(logger).To(gbytes.Say(fmt.Sprintf(`metadata file does not exist.*filename.*%s`, metadataFileName)))
			})
		})

		Context("when the metadata cannot be unmarshaled", func() {
			BeforeEach(func() {
				err := ioutil.WriteFile(metadataFileName, []byte("some-bad-data"), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error", func() {
				err := noBridge.Validate("10.10.40.10/24")
				Expect(err).To(MatchError(ContainSubstring("unmarshalling metadata:")))
			})
		})

		Context("when the subnet cannot be parsed", func() {
			It("returns an error", func() {
				err := noBridge.Validate("%%%%%%%%%%%")
				Expect(err).To(MatchError(ContainSubstring("parsing subnet:")))
			})
		})
	})

})
