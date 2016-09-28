package cli_plugin_test

import (
	"cli-plugin/cli_plugin"
	"cli-plugin/styles"
	"errors"
	"lib/fakes"
	"log"

	"github.com/cloudfoundry/cli/plugin/pluginfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin", func() {
	var (
		policyPlugin      cli_plugin.Plugin
		fakeCliConnection *pluginfakes.FakeCliConnection
		policyClient      *fakes.ExternalPolicyClient
	)

	BeforeEach(func() {
		policyClient = &fakes.ExternalPolicyClient{}
		policyPlugin = cli_plugin.Plugin{
			Styler:       styles.NewGroup(),
			Logger:       log.New(GinkgoWriter, "", 0),
			PolicyClient: policyClient,
		}

		fakeCliConnection = &pluginfakes.FakeCliConnection{}
	})

	Context("when getting the api endpoint fails", func() {
		BeforeEach(func() {
			fakeCliConnection.ApiEndpointReturns("", errors.New("banana"))
		})
		It("returns the error", func() {
			_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"some-command"})
			Expect(err).To(MatchError("getting api endpoint: banana"))
		})
	})

	Context("when checking if ssl is disabled fails", func() {
		BeforeEach(func() {
			fakeCliConnection.IsSSLDisabledReturns(true, errors.New("banana"))
		})
		It("returns the error", func() {
			_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"some-command"})
			Expect(err).To(MatchError("checking if ssl disabled: banana"))
		})
	})

	Describe("ValidateArgs", func() {
		It("returns a struct with validated and converted args", func() {
			argStruct, err := cli_plugin.ValidateArgs(fakeCliConnection, []string{
				"command-arg", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(argStruct).To(Equal(cli_plugin.ValidArgs{
				SourceAppName: "some-app",
				DestAppName:   "some-other-app",
				Protocol:      "tcp",
				Port:          9999,
			}))
		})

		Context("when the flags are in different order", func() {
			It("returns a struct with validated and converted args", func() {
				argStruct, err := cli_plugin.ValidateArgs(fakeCliConnection, []string{
					"command-arg", "some-app", "some-other-app", "--port", "9999", "--protocol", "tcp",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(argStruct).To(Equal(cli_plugin.ValidArgs{
					SourceAppName: "some-app",
					DestAppName:   "some-other-app",
					Protocol:      "tcp",
					Port:          9999,
				}))
			})
		})

		Context("when the port is not an int", func() {
			It("returns a useful error", func() {
				fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{"USAGE:", "banana"}, nil)
				_, err := cli_plugin.ValidateArgs(fakeCliConnection, []string{
					"command-arg", "some-app", "some-other-app", "--protocol", "tcp", "--port", "not-an-int",
				})
				Expect(err).To(MatchError("Incorrect usage. Port is not valid: not-an-int\n\nUSAGE:\nbanana"))
				c := fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)
				Expect(c).To(Equal([]string{"help", "command-arg"}))
			})
			Context("when the cf cli command fails", func() {
				BeforeEach(func() {
					fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{}, errors.New("banana"))
				})
				It("returns the error", func() {
					_, err := cli_plugin.ValidateArgs(fakeCliConnection, []string{
						"command-arg", "some-app", "some-other-app", "--protocol", "tcp", "--port", "not-an-int",
					})
					Expect(err).To(MatchError("cf cli error: banana"))
				})
			})
		})
	})
})
