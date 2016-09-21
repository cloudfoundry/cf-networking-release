package config_test

import (
	"netmon/config"

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
})
