package cli_plugin_test

import (
	"cli-plugin/cli_plugin"
	"cli-plugin/styles"
	"encoding/json"
	"errors"
	"lib/marshal"
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
		srcAppData        plugin_models.GetAppModel
		dstAppData        plugin_models.GetAppModel
	)

	BeforeEach(func() {
		policyPlugin = cli_plugin.Plugin{
			Marshaler:   marshal.MarshalFunc(json.Marshal),
			Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
			Styler:      styles.NewGroup(),
			Logger:      log.New(GinkgoWriter, "", 0),
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

				Expect(output).To(Equal("<BOLD>Source\tDestination\tProtocol\tPort\n<RESET>"))
			})
		})

		Context("when there is a policy and I can resolve the guids", func() {
			It("shows them", func() {
				output, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"list-access"})
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeCliConnection.GetAppsCallCount()).To(Equal(1))
				Expect(fakeCliConnection.CliCommandWithoutTerminalOutputCallCount()).To(Equal(1))
				Expect(fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)).To(Equal([]string{"curl", "/networking/v0/external/policies"}))

				Expect(output).To(Equal("<BOLD>Source\t\tDestination\tProtocol\tPort\n<RESET><CLR_C>some-app<RESET>\t<CLR_C>some-other-app<RESET>\ttcp\t\t9999\n"))
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

				Expect(output).To(Equal("<BOLD>Source\tDestination\tProtocol\tPort\n<RESET>"))
			})
		})

		Context("when getting the username fails", func() {
			BeforeEach(func() {
				fakeCliConnection.UsernameReturns("", errors.New("banana"))
			})

			It("returns an error", func() {
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"list-access"})
				Expect(err).To(MatchError("could not resolve username: banana"))
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

		Context("when getting policies fails", func() {
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

		Context("when the user specifies an app name", func() {
			BeforeEach(func() {
				fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{`{"policies":[
					{"source":{"id":"some-app-guid"},"destination":{"id":"some-other-app-guid","port":9999,"protocol":"tcp"}}
				]}`}, nil)
				fakeCliConnection.GetAppReturns(plugin_models.GetAppModel{
					Guid: "some-app-guid",
					Name: "some-app",
				}, nil)
				fakeCliConnection.GetAppsReturns([]plugin_models.GetAppsModel{
					{Guid: "some-app-guid", Name: "some-app"},
					{Guid: "some-other-app-guid", Name: "some-other-app"},
				}, nil)
			})
			It("filters the call to the policy server", func() {
				output, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"list-access", "--app", "some-app"})
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeCliConnection.GetAppCallCount()).To(Equal(1))
				Expect(fakeCliConnection.GetAppArgsForCall(0)).To(Equal("some-app"))
				Expect(fakeCliConnection.GetAppsCallCount()).To(Equal(1))
				Expect(fakeCliConnection.CliCommandWithoutTerminalOutputCallCount()).To(Equal(1))
				Expect(fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)).To(Equal([]string{"curl", "/networking/v0/external/policies?id=some-app-guid"}))

				Expect(output).To(Equal("<BOLD>Source\t\tDestination\tProtocol\tPort\n<RESET><CLR_C>some-app<RESET>\t<CLR_C>some-other-app<RESET>\ttcp\t\t9999\n"))
			})

			Context("when GetApp fails", func() {
				BeforeEach(func() {
					fakeCliConnection.GetAppReturns(plugin_models.GetAppModel{}, errors.New("ERROR"))
				})
				It("returns the error", func() {
					_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"list-access", "--app", "some-app"})
					Expect(err).To(MatchError("getting app: ERROR"))
				})
			})
		})

		Context("when the user supplies additional arguments", func() {
			It("shows usage", func() {
				fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{"USAGE:", "banana"}, nil)
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"list-access", "some-app"})
				Expect(err).To(MatchError("USAGE:\nbanana"))
				c := fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)
				Expect(c).To(Equal([]string{"help", "list-access"}))
			})
		})
	})

	Describe("AllowCommand", func() {
		BeforeEach(func() {
			fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{"{}\n"}, nil)
		})
		Context("when the command is allow-access", func() {
			It("translates the app names to app guids", func() {
				By("dispatching to the AllowCommand")
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{
					"allow-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999",
				})
				Expect(err).NotTo(HaveOccurred())

				By("translating all the app names to app guids")
				Expect(fakeCliConnection.GetAppCallCount()).To(Equal(2))
				Expect(fakeCliConnection.GetAppArgsForCall(0)).To(Equal("some-app"))
				Expect(fakeCliConnection.GetAppArgsForCall(1)).To(Equal("some-other-app"))

				By("sending a post request to the policy server")
				Expect(fakeCliConnection.CliCommandWithoutTerminalOutputCallCount()).To(Equal(1))
				Expect(fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)).To(Equal([]string{
					"curl", "-X", "POST", "/networking/v0/external/policies", "-d",
					`'{"policies":[{"source":{"id":"some-app-guid"},"destination":{"id":"some-other-app-guid","port":9999,"protocol":"tcp"}}]}'`,
				}))
			})

			Context("when the user supplies incorrect arguments", func() {
				It("shows usage", func() {
					fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{"USAGE:", "banana"}, nil)
					_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "--protocol", "tcp", "some-other-app", "--port", "9999"})
					Expect(err).To(MatchError("USAGE:\nbanana"))
					c := fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)
					Expect(c).To(Equal([]string{"help", "allow-access"}))
				})

				Context("and then when the cf cli command fails", func() {
					BeforeEach(func() {
						fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{}, errors.New("banana"))
					})

					It("returns the error", func() {
						_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access"})
						Expect(err).To(MatchError("cf cli error: banana"))
					})
				})
			})

			Context("when getting the username fails", func() {
				BeforeEach(func() {
					fakeCliConnection.UsernameReturns("", errors.New("banana"))
				})

				It("returns an error", func() {
					_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999"})
					Expect(err).To(MatchError("could not resolve username: banana"))
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

			Context("when the policy server returns a json error", func() {
				BeforeEach(func() {
					fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{`{"error": "banana"}`}, nil)
				})

				It("returns the error and fails the command", func() {
					_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "-protocol", "tcp", "--port", "9999"})
					Expect(err).To(MatchError("error creating policy: banana"))
				})

				Context("when unmarshaling the policy error fails", func() {
					BeforeEach(func() {
						policyPlugin.Unmarshaler = marshal.UnmarshalFunc(func([]byte, interface{}) error {
							return errors.New("banana")
						})
					})
					It("returns the error", func() {
						_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "-protocol", "tcp", "--port", "9999"})
						Expect(err).To(MatchError("error unmarshaling policy response: banana"))
					})
				})
			})

			Context("when the cli curl command fails", func() {
				BeforeEach(func() {
					fakeCliConnection.CliCommandWithoutTerminalOutputReturns(nil, errors.New("blueberry"))
				})

				It("returns a useful error", func() {
					_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"allow-access", "some-app", "some-other-app", "-protocol", "tcp", "--port", "9999"})
					Expect(err).To(MatchError("policy creation failed: blueberry"))
				})
			})
		})
	})

	Describe("DenyCommand", func() {
		BeforeEach(func() {
			fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{"{}\n"}, nil)
		})
		Context("when the policy is found", func() {
			It("removes the policy", func() {
				By("dispatching to the DenyCommand")
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{
					"deny-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999",
				})
				Expect(err).NotTo(HaveOccurred())

				By("translating all the app names to app guids")
				Expect(fakeCliConnection.GetAppCallCount()).To(Equal(2))
				Expect(fakeCliConnection.GetAppArgsForCall(0)).To(Equal("some-app"))
				Expect(fakeCliConnection.GetAppArgsForCall(1)).To(Equal("some-other-app"))

				By("sending a delete request to the policy server")
				Expect(fakeCliConnection.CliCommandWithoutTerminalOutputCallCount()).To(Equal(1))
				Expect(fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)).To(Equal([]string{
					"curl", "-X", "DELETE", "/networking/v0/external/policies", "-d",
					`'{"policies":[{"source":{"id":"some-app-guid"},"destination":{"id":"some-other-app-guid","port":9999,"protocol":"tcp"}}]}'`,
				}))
			})
		})

		Context("when the user supplies incorrect arguments", func() {
			It("shows usage", func() {
				fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{"USAGE:", "banana"}, nil)
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"deny-access", "some-app", "some-other-app", "yet-another-app", "--protocol", "tcp", "--port", "9999"})
				Expect(err).To(MatchError("USAGE:\nbanana"))
				c := fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)
				Expect(c).To(Equal([]string{"help", "deny-access"}))
			})
		})

		Context("when the policies are not marshalable", func() {
			BeforeEach(func() {
				policyPlugin.Marshaler = marshal.MarshalFunc(func(input interface{}) ([]byte, error) {
					return nil, errors.New("banana")
				})
			})

			It("returns a useful error", func() {
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{
					"deny-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999",
				})
				Expect(err).To(MatchError("payload cannot be marshaled: banana"))
			})
		})

		Context("when the cli curl command fails", func() {
			BeforeEach(func() {
				fakeCliConnection.CliCommandWithoutTerminalOutputReturns(nil, errors.New("blueberry"))
			})

			It("returns a useful error", func() {
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{
					"deny-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999",
				})
				Expect(err).To(MatchError("policy deletion failed: blueberry"))
			})
		})

		Context("when getting the username fails", func() {
			BeforeEach(func() {
				fakeCliConnection.UsernameReturns("", errors.New("banana"))
			})

			It("returns an error", func() {
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"deny-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999"})
				Expect(err).To(MatchError("could not resolve username: banana"))
			})
		})

		Context("when the policy server returns a json error", func() {
			BeforeEach(func() {
				fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{`{"error": "banana"}`}, nil)
			})

			It("returns the error and fails the command", func() {
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"deny-access", "some-app", "some-other-app", "-protocol", "tcp", "--port", "9999"})
				Expect(err).To(MatchError("error deleting policy: banana"))
			})

			Context("when unmarshaling the policy error fails", func() {
				BeforeEach(func() {
					policyPlugin.Unmarshaler = marshal.UnmarshalFunc(func([]byte, interface{}) error {
						return errors.New("banana")
					})
				})
				It("returns the error", func() {
					_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{"deny-access", "some-app", "some-other-app", "-protocol", "tcp", "--port", "9999"})
					Expect(err).To(MatchError("error unmarshaling policy response: banana"))
				})
			})
		})
	})

	Describe("Resolving App Names to Guids", func() {
		Context("when there are errors talking to CC", func() {
			It("returns a useful error", func() {
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{
					"deny-access", "bad-access", "some-other-app", "--protocol", "tcp", "--port", "9999",
				})
				Expect(err).To(MatchError("resolving source app: apple"))
			})
		})

		Context("when the source app could not be resolved to a GUID", func() {
			It("returns a useful error", func() {
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{
					"deny-access", "inaccessible-app", "some-other-app", "--protocol", "tcp", "--port", "9999",
				})
				Expect(err).To(MatchError("resolving source app: inaccessible-app not found"))
			})
		})

		Context("when there are errors resolving destination app", func() {
			It("returns a useful error", func() {
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{
					"deny-access", "some-app", "not-some-other-app", "--protocol", "tcp", "--port", "9999",
				})
				Expect(err).To(MatchError("resolving destination app: apple"))
			})
		})

		Context("when the destination app could not be resolved to a GUID", func() {
			It("returns a useful error", func() {
				_, err := policyPlugin.RunWithErrors(fakeCliConnection, []string{
					"deny-access", "some-app", "inaccessible-app", "--protocol", "tcp", "--port", "9999",
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
