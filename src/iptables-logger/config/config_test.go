package config_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"iptables-logger/config"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("New", func() {
		var (
			file *os.File
			err  error
		)

		BeforeEach(func() {
			file, err = ioutil.TempFile(os.TempDir(), "config-")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when config file is valid", func() {
			BeforeEach(func() {
				file.WriteString(`{
					"kernel_log_file": "/var/log/kern.log",
					"container_metadata_file": "/var/vcap/data/container-metadata/store.json",
					"output_log_file": "/var/vcap/sys/log/iptables-logger",
					"host_ip": "1.2.3.4",
					"host_guid": "some-guid"
				}`)
			})
			It("returns the config", func() {
				c, err := config.New(file.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(c.KernelLogFile).To(Equal("/var/log/kern.log"))
				Expect(c.ContainerMetadataFile).To(Equal("/var/vcap/data/container-metadata/store.json"))
				Expect(c.OutputLogFile).To(Equal("/var/vcap/sys/log/iptables-logger"))
				Expect(c.HostIp).To(Equal("1.2.3.4"))
				Expect(c.HostGuid).To(Equal("some-guid"))
			})
		})

		Context("when config file is invalid", func() {
			It("returns the error", func() {
				_, err := config.New("not-exists")
				Expect(err).To(MatchError(ContainSubstring("file does not exist:")))
			})
		})

		Context("when config file is bad format", func() {
			It("returns the error", func() {
				file.WriteString("bad-format")
				_, err = config.New(file.Name())
				Expect(err).To(MatchError(ContainSubstring("parsing config")))
			})
		})

		Context("when config file contents blank", func() {
			It("returns the error", func() {
				_, err = config.New(file.Name())
				Expect(err).To(MatchError(ContainSubstring("parsing config")))
			})
		})

		DescribeTable("when config file is missing a member",
			func(missingFlag, errorMsg string) {
				allData := map[string]interface{}{
					"kernel_log_file":         "/var/log/kern.log",
					"container_metadata_file": "/var/vcap/data/container-metadata/store.json",
					"output_log_file":         "/var/vcap/sys/log/iptables-logger",
					"host_ip":                 "1.2.3.4",
					"host_guid":               "some-guid",
				}
				delete(allData, missingFlag)
				Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())

				_, err = config.New(file.Name())
				Expect(err).To(MatchError(fmt.Sprintf("invalid config: %s", errorMsg)))
			},
			Entry("missing input file", "kernel_log_file", "KernelLogFile: zero value"),
			Entry("missing container metadata file", "container_metadata_file", "ContainerMetadataFile: zero value"),
			Entry("missing output log file", "output_log_file", "OutputLogFile: zero value"),
			Entry("missing host ip", "host_ip", "HostIp: zero value"),
			Entry("missing host guid", "host_guid", "HostGuid: zero value"),
		)
	})
})
