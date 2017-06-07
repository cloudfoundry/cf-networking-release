package config_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"netmon/config"
	"os"

	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("ParseLogLevel", func() {
		DescribeTable("the log levels",
			func(input string, expected lager.LogLevel) {
				cfg := config.Netmon{
					LogLevel: input,
				}
				level, err := cfg.ParseLogLevel()
				Expect(err).NotTo(HaveOccurred())
				Expect(level).To(Equal(expected))
			},
			Entry("DEBUG", "debug", lager.DEBUG),
			Entry("INFO", "Info", lager.INFO),
			Entry("ERROR", "error", lager.ERROR),
			Entry("FATAL", "FATAL", lager.FATAL),
		)

		Context("when the log level cannot be parsed", func() {
			It("returns a useful error", func() {
				cfg := config.Netmon{
					LogLevel: "banana",
				}
				_, err := cfg.ParseLogLevel()
				Expect(err).To(MatchError(`unknown log level "banana"`))
			})
		})
	})

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
			It("returns the config", func() {
				file.WriteString(`{
					"poll_interval": 1234,
					"metron_address": "http://1.2.3.4:1234",
					"interface_name": "eth0",
					"log_level": "debug",
					"log_prefix": "cfnetworking"
				}`)
				c, err := config.New(file.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(c.PollInterval).To(Equal(1234))
				Expect(c.MetronAddress).To(Equal("http://1.2.3.4:1234"))
				Expect(c.InterfaceName).To(Equal("eth0"))
				Expect(c.LogLevel).To(Equal("debug"))
				Expect(c.LogPrefix).To(Equal("cfnetworking"))
			})
		})

		Context("when config file path does not exist", func() {
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
					"poll_interval":  1234,
					"metron_address": "http://1.2.3.4:1234",
					"interface_name": "eth0",
					"log_level":      "debug",
					"log_prefix":     "cfnetworking",
				}
				delete(allData, missingFlag)
				Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())

				_, err = config.New(file.Name())
				Expect(err).To(MatchError(fmt.Sprintf("invalid config: %s", errorMsg)))
			},
			Entry("missing poll interval", "poll_interval", "PollInterval: less than min"),
			Entry("missing metron address", "metron_address", "MetronAddress: zero value"),
			Entry("missing interface name", "interface_name", "InterfaceName: zero value"),
			Entry("missing log prefix", "log_prefix", "LogPrefix: zero value"),
		)
	})
})
