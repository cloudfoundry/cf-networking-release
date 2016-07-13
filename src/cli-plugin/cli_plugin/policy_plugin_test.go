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
		policyPlugin      cli_plugin.Plugin
		fakeCliConnection *pluginfakes.FakeCliConnection
	)

	BeforeEach(func() {
		policyPlugin = cli_plugin.Plugin{
			Marshaler:   marshal.MarshalFunc(json.Marshal),
			Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
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
						plugin.Command{
							Name:     "list-access",
							Alias:    "",
							HelpText: "List policy for direct network traffic from one app to another",
							UsageDetails: plugin.Usage{
								Usage: "cf list-access",
							},
						},
					},
				},
			))
		})
	})

	Describe("ListCommand", func() {
		BeforeEach(func() {
			fakeCliConnection = &pluginfakes.FakeCliConnection{}
			fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{`{"policies":[{"source":{"id":"some-app-guid"},"destination":{"id":"some-other-app-guid","port":9999,"protocol":"tcp"}}]}`}, nil)
			fakeCliConnection.GetAppsReturns([]plugin_models.GetAppsModel{
				{Guid: "some-app-guid", Name: "some-app"},
				{Guid: "some-other-app-guid", Name: "some-other-app"},
			}, nil)
		})

		Context("when there are no policies", func() {
			BeforeEach(func() {
				fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{`{"policies":[]}`}, nil)
			})
			It("shows nothing", func() {
				output, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"list-access"})
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeCliConnection.GetAppsCallCount()).To(Equal(1))
				Expect(fakeCliConnection.CliCommandWithoutTerminalOutputCallCount()).To(Equal(1))
				Expect(fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)).To(Equal([]string{"curl", "/networking/v0/external/policies"}))

				Expect(output).To(Equal("Source\tDestination\tProtocol\tPort\n"))
			})
		})

		Context("when there is a policy and I can resolve the guids", func() {
			It("shows them", func() {
				output, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"list-access"})
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeCliConnection.GetAppsCallCount()).To(Equal(1))
				Expect(fakeCliConnection.CliCommandWithoutTerminalOutputCallCount()).To(Equal(1))
				Expect(fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)).To(Equal([]string{"curl", "/networking/v0/external/policies"}))

				Expect(output).To(Equal("Source\t\tDestination\tProtocol\tPort\nsome-app\tsome-other-app\ttcp\t\t9999\n"))
			})
		})

		Context("when there is a policy but I cannot resolve the guids", func() {
			BeforeEach(func() {
				fakeCliConnection.GetAppsReturns([]plugin_models.GetAppsModel{
					{Guid: "another-guid", Name: "some-app"},
					{Guid: "some-other-app-guid", Name: "some-other-app"},
				}, nil)
			})

			It("shows nothing", func() {
				output, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"list-access"})
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeCliConnection.GetAppsCallCount()).To(Equal(1))
				Expect(fakeCliConnection.CliCommandWithoutTerminalOutputCallCount()).To(Equal(1))
				Expect(fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)).To(Equal([]string{"curl", "/networking/v0/external/policies"}))

				Expect(output).To(Equal("Source\tDestination\tProtocol\tPort\n"))
			})
		})

		Context("when getting the apps fails", func() {
			BeforeEach(func() {
				fakeCliConnection.GetAppsReturns([]plugin_models.GetAppsModel{}, errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"list-access"})
				Expect(err).To(MatchError("getting apps: banana"))
			})
		})

		Context("when getting the apps fails", func() {
			BeforeEach(func() {
				fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{}, errors.New("ERROR"))
			})
			It("returns the error", func() {
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"list-access"})
				Expect(err).To(MatchError("getting policies: ERROR"))
			})
		})

		Context("when the response from the policy server cannot be unmarshalled", func() {
			BeforeEach(func() {
				policyPlugin.Unmarshaler = marshal.UnmarshalFunc(func([]byte, interface{}) error {
					return errors.New("banana")
				})
			})
			It("returns an error", func() {
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"list-access"})
				Expect(err).To(MatchError("unmarshaling: banana"))
			})
		})
	})

	Describe("AllowCommand", func() {
		Context("when the command is allow-access", func() {
			var (
				srcAppData plugin_models.GetAppModel
				dstAppData plugin_models.GetAppModel
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
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999"})
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeCliConnection.GetAppCallCount()).To(Equal(2))
				Expect(fakeCliConnection.GetAppArgsForCall(0)).To(Equal("some-app"))
				Expect(fakeCliConnection.GetAppArgsForCall(1)).To(Equal("some-other-app"))

				Expect(fakeCliConnection.CliCommandWithoutTerminalOutputCallCount()).To(Equal(1))
				Expect(fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)).To(Equal([]string{
					"curl", "-X", "POST", "/networking/v0/external/policies", "-d",
					`'{"policies":[{"source":{"id":"some-app-guid"},"destination":{"id":"some-other-app-guid","port":9999,"protocol":"tcp"}}]}'`,
				}))
			})

			Context("when there are missing args", func() {
				Context("when there are < 2 args", func() {
					It("returns an error", func() {
						_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access"})
						Expect(err).To(MatchError("not enough arguments"))
					})
				})

				Context("when the port is missing", func() {
					It("returns an error", func() {
						_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "--protocol", "tcp"})
						Expect(err).To(MatchError("Requires --port PORT as argument."))
					})
				})

				Context("when the protocol is missing", func() {
					It("returns an error", func() {
						_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "--port", "9999"})
						Expect(err).To(MatchError("Requires --protocol PROTOCOL as argument."))
					})
				})
			})

			Context("when the args are in a different order", func() {
				It("parses them correctly", func() {
					_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "--port", "9999", "--protocol", "tcp"})
					Expect(err).NotTo(HaveOccurred())
					Expect(fakeCliConnection.GetAppCallCount()).To(Equal(2))
					Expect(fakeCliConnection.GetAppArgsForCall(0)).To(Equal("some-app"))
					Expect(fakeCliConnection.GetAppArgsForCall(1)).To(Equal("some-other-app"))

					Expect(fakeCliConnection.CliCommandWithoutTerminalOutputCallCount()).To(Equal(1))
					Expect(fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)).To(Equal([]string{
						"curl", "-X", "POST", "/networking/v0/external/policies", "-d",
						`'{"policies":[{"source":{"id":"some-app-guid"},"destination":{"id":"some-other-app-guid","port":9999,"protocol":"tcp"}}]}'`,
					}))
				})
			})

			Context("when there are errors talking to CC", func() {
				It("returns a useful error", func() {
					_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "bad-access", "some-other-app", "--protocol", "tcp", "--port", "9999"})
					Expect(err).To(MatchError("resolving source app: apple"))
				})
			})

			Context("when the source app could not be resolved to a GUID", func() {
				It("returns a useful error", func() {
					_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "inaccessible-app", "some-other-app", "--protocol", "tcp", "--port", "9999"})
					Expect(err).To(MatchError("resolving source app: inaccessible-app not found"))
				})
			})

			Context("when there are errors resolving destination app", func() {
				It("returns a useful error", func() {
					_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "not-some-other-app", "--protocol", "tcp", "--port", "9999"})
					Expect(err).To(MatchError("resolving destination app: apple"))
				})
			})

			Context("when the destination app could not be resolved to a GUID", func() {
				It("returns a useful error", func() {
					_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "inaccessible-app", "--protocol", "tcp", "--port", "9999"})
					Expect(err).To(MatchError("resolving destination app: inaccessible-app not found"))
				})
			})

			Context("when the port is not an int", func() {
				It("returns a useful error", func() {
					_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "not-an-int"})
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
					_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999"})
					Expect(err).To(MatchError("payload cannot be marshaled: banana"))
				})
			})

			Context("when the cli curl command fails", func() {
				BeforeEach(func() {
					fakeCliConnection.CliCommandWithoutTerminalOutputReturns(nil, errors.New("blueberry"))
				})

				It("returns a useful error", func() {
					_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999"})
					Expect(err).To(MatchError("policy creation failed: blueberry"))
				})
			})
		})
	})
})
