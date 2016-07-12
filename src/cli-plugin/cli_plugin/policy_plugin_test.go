package cli_plugin_test

import (
	"cli-plugin/cli_plugin"
	"encoding/json"
	"errors"
	"lib/marshal"

	"github.com/cloudfoundry/cli/plugin"
	"github.com/cloudfoundry/cli/plugin/models"
	"github.com/cloudfoundry/cli/plugin/pluginfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin", func() {
	var (
		policyPlugin cli_plugin.Plugin
	)

	BeforeEach(func() {
		policyPlugin = cli_plugin.Plugin{
			Marshaler: marshal.MarshalFunc(json.Marshal),
		}
	})

	Describe("GetMetadata", func() {
		It("responds with its metadata", func() {
			Expect(policyPlugin.GetMetadata()).To(Equal(
				plugin.PluginMetadata{
					Name: "network-policy",
					Version: plugin.VersionType{
						Major: 0,
						Minor: 0,
					},
					MinCliVersion: plugin.VersionType{
						Major: 6,
						Minor: 15,
					},
					Commands: []plugin.Command{
						plugin.Command{
							Name:     "allow-access",
							HelpText: "Allow direct network traffic from one app to another",
							UsageDetails: plugin.Usage{
								Usage: "cf allow-access SOURCE_APP DESTINATION_APP --protocol <tcp|udp> --port [1-65535]",
							},
						},
					},
				},
			))
		})
	})

	Describe("RunWithErrors", func() {
		Context("when the command is allow-access", func() {
			var (
				srcAppData        plugin_models.GetAppModel
				dstAppData        plugin_models.GetAppModel
				fakeCliConnection *pluginfakes.FakeCliConnection
			)

			BeforeEach(func() {
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

			It("translates the app names to app guids", func() {
				err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999"})
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeCliConnection.GetAppCallCount()).To(Equal(2))
				Expect(fakeCliConnection.GetAppArgsForCall(0)).To(Equal("some-app"))
				Expect(fakeCliConnection.GetAppArgsForCall(1)).To(Equal("some-other-app"))

				Expect(fakeCliConnection.CliCommandCallCount()).To(Equal(1))
				Expect(fakeCliConnection.CliCommandArgsForCall(0)).To(Equal([]string{
					"curl", "-X", "POST", "/networking/v0/external/policies", "-d",
					`'{"policies":[{"source":{"id":"some-app-guid"},"destination":{"id":"some-other-app-guid","port":9999,"protocol":"tcp"}}]}'`,
				}))
			})

			Context("when there are missing args", func() {
				Context("when there are < 2 args", func() {
					It("returns an error", func() {
						err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access"})
						Expect(err).To(MatchError("not enough arguments"))
					})
				})

				Context("when the port is missing", func() {
					It("returns an error", func() {
						err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "--protocol", "tcp"})
						Expect(err).To(MatchError("Requires --port PORT as argument."))
					})
				})

				Context("when the protocol is missing", func() {
					It("returns an error", func() {
						err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "--port", "9999"})
						Expect(err).To(MatchError("Requires --protocol PROTOCOL as argument."))
					})
				})
			})

			Context("when the args are in a different order", func() {
				It("parses them correctly", func() {
					err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "--port", "9999", "--protocol", "tcp"})
					Expect(err).NotTo(HaveOccurred())
					Expect(fakeCliConnection.GetAppCallCount()).To(Equal(2))
					Expect(fakeCliConnection.GetAppArgsForCall(0)).To(Equal("some-app"))
					Expect(fakeCliConnection.GetAppArgsForCall(1)).To(Equal("some-other-app"))

					Expect(fakeCliConnection.CliCommandCallCount()).To(Equal(1))
					Expect(fakeCliConnection.CliCommandArgsForCall(0)).To(Equal([]string{
						"curl", "-X", "POST", "/networking/v0/external/policies", "-d",
						`'{"policies":[{"source":{"id":"some-app-guid"},"destination":{"id":"some-other-app-guid","port":9999,"protocol":"tcp"}}]}'`,
					}))
				})
			})

			Context("when there are errors talking to CC", func() {
				It("returns a useful error", func() {
					err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "bad-access", "some-other-app", "--protocol", "tcp", "--port", "9999"})
					Expect(err).To(MatchError("resolving source app: apple"))
				})
			})

			Context("when the source app could not be resolved to a GUID", func() {
				It("returns a useful error", func() {
					err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "inaccessible-app", "some-other-app", "--protocol", "tcp", "--port", "9999"})
					Expect(err).To(MatchError("resolving source app: inaccessible-app not found"))
				})
			})

			Context("when there are errors resolving destination app", func() {
				It("returns a useful error", func() {
					err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "not-some-other-app", "--protocol", "tcp", "--port", "9999"})
					Expect(err).To(MatchError("resolving destination app: apple"))
				})
			})

			Context("when the destination app could not be resolved to a GUID", func() {
				It("returns a useful error", func() {
					err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "inaccessible-app", "--protocol", "tcp", "--port", "9999"})
					Expect(err).To(MatchError("resolving destination app: inaccessible-app not found"))
				})
			})

			Context("when the port is not an int", func() {
				It("returns a useful error", func() {
					err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "not-an-int"})
					Expect(err).To(MatchError(`port is not valid: not-an-int`))
				})
			})

			Context("when the policies are not marshalable", func() {
				BeforeEach(func() {
					policyPlugin.Marshaler = marshal.MarshalFunc(func(input interface{}) ([]byte, error) {
						return nil, errors.New("banana")
					})
				})

				It("returns a useful error", func() {
					err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999"})
					Expect(err).To(MatchError("payload cannot be marshaled: banana"))
				})
			})

			Context("when the cli curl command fails", func() {
				BeforeEach(func() {
					fakeCliConnection.CliCommandReturns(nil, errors.New("blueberry"))
				})

				It("returns a useful error", func() {
					err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999"})
					Expect(err).To(MatchError("policy creation failed: blueberry"))
				})
			})
		})
	})
})
