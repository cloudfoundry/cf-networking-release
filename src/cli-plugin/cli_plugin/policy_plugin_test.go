package cli_plugin_test

import (
	"cli-plugin/cli_plugin"
	"cli-plugin/styles"
	"errors"
	"lib/fakes"
	"log"

	"github.com/cloudfoundry/cli/plugin/models"
	"github.com/cloudfoundry/cli/plugin/pluginfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin", func() {
	var (
		policyPlugin      cli_plugin.Plugin
		fakeCliConnection *pluginfakes.FakeCliConnection
		policyClient      *fakes.ExternalPolicyClient
		srcAppData        plugin_models.GetAppModel
		dstAppData        plugin_models.GetAppModel
	)

	BeforeEach(func() {
		policyClient = &fakes.ExternalPolicyClient{}
		policyPlugin = cli_plugin.Plugin{
			Styler:       styles.NewGroup(),
			Logger:       log.New(GinkgoWriter, "", 0),
			PolicyClient: policyClient,
		}

		srcAppData = plugin_models.GetAppModel{
			Name: "some-app",
			Guid: "some-app-guid",
		}
		dstAppData = plugin_models.GetAppModel{
			Name: "some-other-app",
			Guid: "some-other-app-guid",
		}
		fakeCliConnection = &pluginfakes.FakeCliConnection{}
		fakeCliConnection.GetAppStub = func(name string) (plugin_models.GetAppModel, error) {
			switch name {
			case "some-app":
				return srcAppData, nil
			case "some-other-app":
				return dstAppData, nil
			case "inaccessible-app":
				return plugin_models.GetAppModel{}, nil
			default:
				return plugin_models.GetAppModel{}, errors.New("apple")
			}
		}
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

	Describe("Resolving App Names to Guids", func() {
		Context("when there are errors talking to CC", func() {
			It("returns a useful error", func() {
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{
					"access-deny", "bad-access", "some-other-app", "--protocol", "tcp", "--port", "9999",
				})
				Expect(err).To(MatchError("resolving source app: apple"))
			})
		})

		Context("when the source app could not be resolved to a GUID", func() {
			It("returns a useful error", func() {
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{
					"access-deny", "inaccessible-app", "some-other-app", "--protocol", "tcp", "--port", "9999",
				})
				Expect(err).To(MatchError("resolving source app: inaccessible-app not found"))
			})
		})

		Context("when there are errors resolving destination app", func() {
			It("returns a useful error", func() {
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{
					"access-deny", "some-app", "not-some-other-app", "--protocol", "tcp", "--port", "9999",
				})
				Expect(err).To(MatchError("resolving destination app: apple"))
			})
		})

		Context("when the destination app could not be resolved to a GUID", func() {
			It("returns a useful error", func() {
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{
					"access-deny", "some-app", "inaccessible-app", "--protocol", "tcp", "--port", "9999",
				})
				Expect(err).To(MatchError("resolving destination app: inaccessible-app not found"))
			})
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
