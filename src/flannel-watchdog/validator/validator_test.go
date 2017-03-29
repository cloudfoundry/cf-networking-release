package validator_test

import (
	"encoding/json"
	"flannel-watchdog/validator"
	"io/ioutil"
	"lib/datastore"
	"lib/filelock"
	"lib/serial"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validator", func() {
	Describe("Validate", func() {
		var (
			subnetValidator  *validator.Validator
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

			store := &datastore.Store{
				Serializer: &serial.Serial{},
				Locker: &filelock.Locker{
					Path: metadataFileName,
				},
			}
			subnetValidator = &validator.Validator{
				Store: store,
			}
		})

		Context("when the container ips fall within the subnet env range", func() {
			It("returns successfully", func() {
				err := subnetValidator.Validate("10.244.40.0/12")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the container ips are outside subnet env range", func() {
			It("returns an error", func() {
				err := subnetValidator.Validate("10.10.40.10/24")
				Expect(err).To(MatchError(`This cell must be restarted (run "bosh restart <job>").  Flannel is out of sync with current containers.`))
			})
		})

		Context("when the metadata file is empty", func() {
			BeforeEach(func() {
				err := ioutil.WriteFile(metadataFileName, []byte(""), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())
			})

			It("return nil", func() {
				err := subnetValidator.Validate("10.10.40.10/24")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the metadata cannot be unmarshaled", func() {
			BeforeEach(func() {
				err := ioutil.WriteFile(metadataFileName, []byte("some-bad-data"), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error", func() {
				err := subnetValidator.Validate("10.10.40.10/24")
				Expect(err).To(MatchError(ContainSubstring("reading metadata:")))
			})
		})

		Context("when the subnet cannot be parsed", func() {
			It("returns an error", func() {
				err := subnetValidator.Validate("%%%%%%%%%%%")
				Expect(err).To(MatchError(ContainSubstring("parsing subnet:")))
			})
		})
	})

})
