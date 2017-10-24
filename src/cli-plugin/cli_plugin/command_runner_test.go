package cli_plugin_test

import (
	"cli-plugin/cli_plugin"
	"cli-plugin/styles"
	"errors"
	"lib/fakes"
	"log"
	"policy-server/api/api_v0"

	"code.cloudfoundry.org/cli/plugin/models"
	"code.cloudfoundry.org/cli/plugin/pluginfakes"

	"policy-server/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CommandRunner", func() {
	var (
		policyClient      *fakes.ExternalPolicyClient
		fakeCliConnection *pluginfakes.FakeCliConnection
		runner            cli_plugin.CommandRunner
		srcAppData        plugin_models.GetAppModel
		dstAppData        plugin_models.GetAppModel
	)

	BeforeEach(func() {
		policyClient = &fakes.ExternalPolicyClient{}
		fakeCliConnection = &pluginfakes.FakeCliConnection{}
		runner = cli_plugin.CommandRunner{
			Styler:        styles.NewGroup(),
			Logger:        log.New(GinkgoWriter, "", 0),
			PolicyClient:  policyClient,
			CliConnection: fakeCliConnection,
			Args:          []string{},
		}
		fakeCliConnection.AccessTokenReturns("some-token", nil)
		fakeCliConnection.GetAppsReturns([]plugin_models.GetAppsModel{
			{Guid: "some-app-guid", Name: "some-app"},
			{Guid: "some-other-app-guid", Name: "some-other-app"},
		}, nil)
		srcAppData = plugin_models.GetAppModel{
			Name: "some-app",
			Guid: "some-app-guid",
		}
		dstAppData = plugin_models.GetAppModel{
			Name: "some-other-app",
			Guid: "some-other-app-guid",
		}
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

	Describe("List", func() {
		BeforeEach(func() {
			runner.Args = []string{"list-access"}
		})

		Context("when we fail to determine an api version", func() {
			BeforeEach(func() {
				policyClient.GetAPIVersionReturns(0, errors.New("pineapple"))
			})

			It("returns a useful error message", func() {
				_, err := runner.List()
				Expect(err).To(MatchError("could not determine networking API version: pineapple"))
			})
		})

		Context("when api version is too high", func() {
			BeforeEach(func() {
				policyClient.GetAPIVersionReturns(2, nil)
			})

			It("returns a useful error message", func() {
				_, err := runner.List()
				Expect(err).To(MatchError("networking API version 2 not supported. consider using cf-cli native commands."))
			})
		})

		Context("when targeting a v0 environment", func() {
			BeforeEach(func() {
				policyClient.GetPoliciesV0Returns([]api_v0.Policy{
					api_v0.Policy{
						Source: api_v0.Source{ID: "some-app-guid"},
						Destination: api_v0.Destination{
							ID:       "some-other-app-guid",
							Port:     9999,
							Protocol: "tcp",
						},
					},
				}, nil)
				policyClient.GetAPIVersionReturns(0, nil)
			})

			Context("when there is a policy and I can resolve the guids", func() {
				It("shows them", func() {
					output, err := runner.List()
					Expect(err).NotTo(HaveOccurred())
					Expect(policyClient.GetAPIVersionCallCount()).To(Equal(1))
					Expect(policyClient.GetPoliciesV0CallCount()).To(Equal(1))
					Expect(policyClient.GetPoliciesV0ArgsForCall(0)).To(Equal("some-token"))
					Expect(fakeCliConnection.GetAppsCallCount()).To(Equal(1))

					Expect(output).To(Equal("<BOLD>Source\t\tDestination\tProtocol\tPort\n<RESET><CLR_C>some-app<RESET>\t<CLR_C>some-other-app<RESET>\ttcp\t\t9999\n"))
				})
			})

			Context("when the user specifies an app name", func() {
				BeforeEach(func() {
					policyClient.GetPoliciesV0ByIDReturns([]api_v0.Policy{
						api_v0.Policy{
							Source: api_v0.Source{ID: "some-app-guid"},
							Destination: api_v0.Destination{
								ID:       "some-other-app-guid",
								Port:     9999,
								Protocol: "tcp",
							},
						},
					}, nil)
					fakeCliConnection.GetAppReturns(plugin_models.GetAppModel{
						Guid: "some-app-guid",
						Name: "some-app",
					}, nil)
					fakeCliConnection.GetAppsReturns([]plugin_models.GetAppsModel{
						{Guid: "some-app-guid", Name: "some-app"},
						{Guid: "some-other-app-guid", Name: "some-other-app"},
					}, nil)
					runner.Args = []string{"list-access", "--app", "some-app"}
				})
				It("filters the call to the policy server", func() {
					output, err := runner.List()
					Expect(err).NotTo(HaveOccurred())
					Expect(output).To(Equal("<BOLD>Source\t\tDestination\tProtocol\tPort\n<RESET><CLR_C>some-app<RESET>\t<CLR_C>some-other-app<RESET>\ttcp\t\t9999\n"))

					Expect(fakeCliConnection.GetAppCallCount()).To(Equal(1))
					Expect(fakeCliConnection.GetAppArgsForCall(0)).To(Equal("some-app"))
					Expect(fakeCliConnection.GetAppsCallCount()).To(Equal(1))

					Expect(policyClient.GetPoliciesV0ByIDCallCount()).To(Equal(1))
					token, ids := policyClient.GetPoliciesV0ByIDArgsForCall(0)
					Expect(token).To(Equal("some-token"))
					Expect(ids).To(ConsistOf("some-app-guid"))
				})
			})

			Context("when gettings the policies fails", func() {
				BeforeEach(func() {
					policyClient.GetPoliciesV0Returns([]api_v0.Policy{}, errors.New("banana"))
				})

				It("returns a useful error message", func() {
					_, err := runner.List()
					Expect(err).To(MatchError("getting policies: banana"))
				})
			})

			Context("when gettings the policies by ID fails", func() {
				BeforeEach(func() {
					policyClient.GetPoliciesV0ByIDReturns([]api_v0.Policy{}, errors.New("banana"))
					runner.Args = []string{"list-access", "--app", "some-app"}
				})

				It("returns a useful error message", func() {
					_, err := runner.List()
					Expect(err).To(MatchError("getting policies by id: banana"))
				})
			})
		})

		Context("when targeting a v1 environment", func() {
			BeforeEach(func() {
				policyClient.GetPoliciesReturns([]api.Policy{
					api.Policy{
						Source: api.Source{ID: "some-app-guid"},
						Destination: api.Destination{
							ID:       "some-other-app-guid",
							Ports:    api.Ports{Start: 9999, End: 1000},
							Protocol: "tcp",
						},
					},
				}, nil)

				policyClient.GetAPIVersionReturns(1, nil)
			})

			Context("when there is a policy and I can resolve the guids", func() {
				It("shows them", func() {
					output, err := runner.List()
					Expect(err).NotTo(HaveOccurred())

					Expect(policyClient.GetPoliciesCallCount()).To(Equal(1))
					Expect(policyClient.GetPoliciesArgsForCall(0)).To(Equal("some-token"))
					Expect(fakeCliConnection.GetAppsCallCount()).To(Equal(1))

					Expect(output).To(Equal("<BOLD>Source\t\tDestination\tProtocol\tPorts\n<RESET><CLR_C>some-app<RESET>\t<CLR_C>some-other-app<RESET>\ttcp\t\t9999-1000\n"))
				})
			})

			Context("when there are no policies", func() {
				BeforeEach(func() {
					policyClient.GetPoliciesReturns([]api.Policy{}, nil)
				})
				It("shows nothing", func() {
					output, err := runner.List()
					Expect(err).NotTo(HaveOccurred())

					Expect(policyClient.GetPoliciesCallCount()).To(Equal(1))
					Expect(policyClient.GetPoliciesArgsForCall(0)).To(Equal("some-token"))
					Expect(fakeCliConnection.GetAppsCallCount()).To(Equal(1))

					Expect(output).To(Equal("<BOLD>Source\tDestination\tProtocol\tPorts\n<RESET>"))
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
					output, err := runner.List()
					Expect(err).NotTo(HaveOccurred())

					Expect(policyClient.GetPoliciesCallCount()).To(Equal(1))
					Expect(policyClient.GetPoliciesArgsForCall(0)).To(Equal("some-token"))
					Expect(fakeCliConnection.GetAppsCallCount()).To(Equal(1))

					Expect(output).To(Equal("<BOLD>Source\tDestination\tProtocol\tPorts\n<RESET>"))
				})
			})

			Context("when getting the username fails", func() {
				BeforeEach(func() {
					fakeCliConnection.UsernameReturns("", errors.New("banana"))
				})

				It("returns an error", func() {
					_, err := runner.List()
					Expect(err).To(MatchError("could not resolve username: banana"))
				})
			})

			Context("when getting apps fails", func() {
				BeforeEach(func() {
					fakeCliConnection.GetAppsReturns([]plugin_models.GetAppsModel{}, errors.New("banana"))
				})
				It("returns the error", func() {
					_, err := runner.List()
					Expect(err).To(MatchError("getting apps: banana"))
				})
			})

			Context("when getting policies fails", func() {
				BeforeEach(func() {
					policyClient.GetPoliciesReturns(nil, errors.New("banana"))
				})
				It("wraps the error in a more helpful message", func() {
					_, err := runner.List()
					Expect(err).To(MatchError("getting policies: banana"))
				})
			})

			Context("when getting access token fails", func() {
				BeforeEach(func() {
					fakeCliConnection.AccessTokenReturns("", errors.New("banana"))
				})
				It("returns the error", func() {
					_, err := runner.List()
					Expect(err).To(MatchError("getting access token: banana"))
				})
			})

			Context("when the user specifies an app name", func() {
				BeforeEach(func() {
					policyClient.GetPoliciesByIDReturns([]api.Policy{
						api.Policy{Source: api.Source{ID: "some-app-guid"}, Destination: api.Destination{ID: "some-other-app-guid", Ports: api.Ports{Start: 9999, End: 1000}, Protocol: "tcp"}},
					}, nil)
					fakeCliConnection.GetAppReturns(plugin_models.GetAppModel{
						Guid: "some-app-guid",
						Name: "some-app",
					}, nil)
					fakeCliConnection.GetAppsReturns([]plugin_models.GetAppsModel{
						{Guid: "some-app-guid", Name: "some-app"},
						{Guid: "some-other-app-guid", Name: "some-other-app"},
					}, nil)
					runner.Args = []string{"list-access", "--app", "some-app"}
				})
				It("filters the call to the policy server", func() {
					output, err := runner.List()
					Expect(err).NotTo(HaveOccurred())
					Expect(output).To(Equal("<BOLD>Source\t\tDestination\tProtocol\tPorts\n<RESET><CLR_C>some-app<RESET>\t<CLR_C>some-other-app<RESET>\ttcp\t\t9999-1000\n"))

					Expect(fakeCliConnection.GetAppCallCount()).To(Equal(1))
					Expect(fakeCliConnection.GetAppArgsForCall(0)).To(Equal("some-app"))
					Expect(fakeCliConnection.GetAppsCallCount()).To(Equal(1))

					Expect(policyClient.GetPoliciesByIDCallCount()).To(Equal(1))
					token, ids := policyClient.GetPoliciesByIDArgsForCall(0)
					Expect(token).To(Equal("some-token"))
					Expect(ids).To(ConsistOf("some-app-guid"))
				})

				Context("when GetApp fails", func() {
					BeforeEach(func() {
						fakeCliConnection.GetAppReturns(plugin_models.GetAppModel{}, errors.New("ERROR"))
					})
					It("returns the error", func() {
						_, err := runner.List()
						Expect(err).To(MatchError("getting app: ERROR"))
					})
				})

				Context("when getting policies by ID fails", func() {
					BeforeEach(func() {
						policyClient.GetPoliciesByIDReturns(nil, errors.New("banana"))
					})
					It("wraps the error in a more helpful message", func() {
						_, err := runner.List()
						Expect(err).To(MatchError("getting policies by id: banana"))
					})
				})

			})

			Context("when the user supplies additional arguments", func() {
				BeforeEach(func() {
					runner.Args = []string{"list-access", "some-app"}
				})
				It("shows usage", func() {
					fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{"USAGE:", "banana"}, nil)
					_, err := runner.List()
					Expect(err).To(MatchError("Incorrect usage. \n\nUSAGE:\nbanana"))
					c := fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)
					Expect(c).To(Equal([]string{"help", "list-access"}))
				})
			})
		})
	})

	Describe("Allow", func() {
		Context("when using a port range", func() {
			BeforeEach(func() {
				policyClient.GetAPIVersionReturns(1, nil)
				runner.Args = []string{"allow-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9998-9999"}
			})

			Context("when the command is allow-access", func() {
				It("translates the app names to app guids", func() {
					_, err := runner.Allow()
					Expect(err).NotTo(HaveOccurred())

					Expect(fakeCliConnection.GetAppCallCount()).To(Equal(2))
					Expect(fakeCliConnection.GetAppArgsForCall(0)).To(Equal("some-app"))
					Expect(fakeCliConnection.GetAppArgsForCall(1)).To(Equal("some-other-app"))

					Expect(policyClient.GetAPIVersionCallCount()).To(Equal(1))
					Expect(policyClient.AddPoliciesCallCount()).To(Equal(1))
					token, policies := policyClient.AddPoliciesArgsForCall(0)
					Expect(token).To(Equal("some-token"))
					Expect(policies).To(ConsistOf(api.Policy{
						Source: api.Source{ID: "some-app-guid"},
						Destination: api.Destination{
							ID: "some-other-app-guid",
							Ports: api.Ports{
								Start: 9998,
								End:   9999,
							},
							Protocol: "tcp"}}))
				})

				Context("when targeting a v0 environment", func() {
					BeforeEach(func() {
						policyClient.GetAPIVersionReturns(0, nil)
					})

					It("returns a useful error message", func() {
						_, err := runner.Allow()
						Expect(err).To(MatchError("the environment targeted is v0. port ranges are supported in v1+"))
					})
				})

				Context("when adding the policies fails", func() {
					BeforeEach(func() {
						policyClient.AddPoliciesReturns(errors.New("banana"))
					})
					It("wraps the error in a more helpful message", func() {
						_, err := runner.Allow()
						Expect(err).To(MatchError("adding policies: banana"))
					})
				})

				Context("when getting the access token fails", func() {
					BeforeEach(func() {
						fakeCliConnection.AccessTokenReturns("", errors.New("banana"))
					})
					It("returns the error", func() {
						_, err := runner.Allow()
						Expect(err).To(MatchError("getting access token: banana"))
					})
				})

				Context("when the user supplies incorrect arguments", func() {
					BeforeEach(func() {
						runner.Args = []string{"allow-access", "some-app", "--protocol", "tcp", "some-other-app", "--port", "9998-9999"}
					})
					It("shows usage", func() {
						fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{"USAGE:", "banana"}, nil)
						_, err := runner.Allow()
						Expect(err).To(MatchError("Incorrect usage. \n\nUSAGE:\nbanana"))
						c := fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)
						Expect(c).To(Equal([]string{"help", "allow-access"}))
					})

					Context("and then when the cf cli command fails", func() {
						BeforeEach(func() {
							fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{}, errors.New("banana"))
						})
						It("returns the error", func() {
							_, err := runner.Allow()
							Expect(err).To(MatchError("cf cli error: banana"))
						})
					})
				})

				Context("when getting the username fails", func() {
					BeforeEach(func() {
						fakeCliConnection.UsernameReturns("", errors.New("banana"))
					})
					It("returns an error", func() {
						_, err := runner.Allow()
						Expect(err).To(MatchError("could not resolve username: banana"))
					})
				})
			})
		})

		Context("when using a single port", func() {
			BeforeEach(func() {
				policyClient.GetAPIVersionReturns(0, nil)
				runner.Args = []string{"allow-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999"}
			})

			Context("when the command is allow-access", func() {
				It("translates the app names to app guids", func() {
					_, err := runner.Allow()
					Expect(err).NotTo(HaveOccurred())

					Expect(fakeCliConnection.GetAppCallCount()).To(Equal(2))
					Expect(fakeCliConnection.GetAppArgsForCall(0)).To(Equal("some-app"))
					Expect(fakeCliConnection.GetAppArgsForCall(1)).To(Equal("some-other-app"))

					Expect(policyClient.AddPoliciesV0CallCount()).To(Equal(1))
					token, policies := policyClient.AddPoliciesV0ArgsForCall(0)
					Expect(token).To(Equal("some-token"))
					Expect(policies).To(ConsistOf(api_v0.Policy{
						Source:      api_v0.Source{ID: "some-app-guid"},
						Destination: api_v0.Destination{ID: "some-other-app-guid", Port: 9999, Protocol: "tcp"}}))
				})

				Context("when the targeted environment is v1", func() {
					BeforeEach(func() {
						policyClient.GetAPIVersionReturns(1, nil)
					})
					It("uses the v1 endpoint", func() {
						_, err := runner.Allow()
						Expect(err).NotTo(HaveOccurred())

						Expect(policyClient.AddPoliciesCallCount()).To(Equal(1))
						Expect(policyClient.AddPoliciesV0CallCount()).To(Equal(0))
					})
				})

				Context("when adding the policies fails", func() {
					BeforeEach(func() {
						policyClient.AddPoliciesV0Returns(errors.New("banana"))
					})
					It("wraps the error in a more helpful message", func() {
						_, err := runner.Allow()
						Expect(err).To(MatchError("adding policies: banana"))
					})
				})

				Context("when getting the access token fails", func() {
					BeforeEach(func() {
						fakeCliConnection.AccessTokenReturns("", errors.New("banana"))
					})
					It("returns the error", func() {
						_, err := runner.Allow()
						Expect(err).To(MatchError("getting access token: banana"))
					})
				})

				Context("when the user supplies incorrect arguments", func() {
					BeforeEach(func() {
						runner.Args = []string{"allow-access", "some-app", "--protocol", "tcp", "some-other-app", "--port", "9999"}
					})
					It("shows usage", func() {
						fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{"USAGE:", "banana"}, nil)
						_, err := runner.Allow()
						Expect(err).To(MatchError("Incorrect usage. \n\nUSAGE:\nbanana"))
						c := fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)
						Expect(c).To(Equal([]string{"help", "allow-access"}))
					})

					Context("and then when the cf cli command fails", func() {
						BeforeEach(func() {
							fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{}, errors.New("banana"))
						})
						It("returns the error", func() {
							_, err := runner.Allow()
							Expect(err).To(MatchError("cf cli error: banana"))
						})
					})
				})

				Context("when getting the username fails", func() {
					BeforeEach(func() {
						fakeCliConnection.UsernameReturns("", errors.New("banana"))
					})
					It("returns an error", func() {
						_, err := runner.Allow()
						Expect(err).To(MatchError("could not resolve username: banana"))
					})
				})
			})

			Context("when we fail to determine an api version", func() {
				BeforeEach(func() {
					policyClient.GetAPIVersionReturns(0, errors.New("pineapple"))
					runner.Args = []string{"allow-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999"}
				})

				It("returns a useful error message", func() {
					_, err := runner.Allow()
					Expect(err).To(MatchError("could not determine networking API version: pineapple"))
				})
			})

			Context("when api version is too high", func() {
				BeforeEach(func() {
					policyClient.GetAPIVersionReturns(2, nil)
					runner.Args = []string{"allow-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999"}
				})

				It("returns a useful error message", func() {
					_, err := runner.Allow()
					Expect(err).To(MatchError("networking API version 2 not supported. consider using cf-cli native commands."))
				})
			})
		})
	})

	Describe("Remove", func() {
		Context("when using a port range", func() {
			BeforeEach(func() {
				policyClient.GetAPIVersionReturns(1, nil)
				runner.Args = []string{"remove-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "8888-9999"}
			})

			Context("when the policy is found", func() {
				It("removes the policy", func() {
					_, err := runner.Remove()
					Expect(err).NotTo(HaveOccurred())

					Expect(fakeCliConnection.GetAppCallCount()).To(Equal(2))
					Expect(fakeCliConnection.GetAppArgsForCall(0)).To(Equal("some-app"))
					Expect(fakeCliConnection.GetAppArgsForCall(1)).To(Equal("some-other-app"))

					Expect(policyClient.GetAPIVersionCallCount()).To(Equal(1))
					Expect(policyClient.DeletePoliciesCallCount()).To(Equal(1))
					token, policies := policyClient.DeletePoliciesArgsForCall(0)
					Expect(token).To(Equal("some-token"))
					Expect(policies).To(ConsistOf(api.Policy{
						Source:      api.Source{ID: "some-app-guid"},
						Destination: api.Destination{ID: "some-other-app-guid", Ports: api.Ports{Start: 8888, End: 9999}, Protocol: "tcp"}}))
				})
			})

			Context("when targeting a v0 environment", func() {
				BeforeEach(func() {
					policyClient.GetAPIVersionReturns(0, nil)
				})

				It("returns a useful error message", func() {
					_, err := runner.Remove()
					Expect(err).To(MatchError("the environment targeted is v0. port ranges are supported in v1+"))
				})
			})

			Context("when the user supplies incorrect arguments", func() {
				Context("when there are too many leading positional arguments", func() {
					It("shows usage", func() {
						runner.Args = []string{"remove-access", "some-app", "some-other-app", "yet-another-app", "--protocol", "tcp", "--port", "8888-9999"}
						fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{"USAGE:", "banana"}, nil)
						_, err := runner.Remove()
						Expect(err).To(MatchError("Incorrect usage. \n\nUSAGE:\nbanana"))
						c := fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)
						Expect(c).To(Equal([]string{"help", "remove-access"}))
					})
				})
				Context("when there are extra positional arguments after the flag args", func() {
					It("shows usage", func() {
						runner.Args = []string{"remove-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999", "something-else"}
						fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{"USAGE:", "banana"}, nil)
						_, err := runner.Remove()
						Expect(err).To(MatchError("Incorrect usage. \n\nUSAGE:\nbanana"))
						c := fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)
						Expect(c).To(Equal([]string{"help", "remove-access"}))
					})
				})
				Context("when one of the flags is misspelled", func() {
					It("shows usage", func() {
						runner.Args = []string{"remove-access", "some-app", "some-other-app", "--protocol", "tcp", "--poooort", "8888-9999"}
						fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{"USAGE:", "banana"}, nil)
						_, err := runner.Remove()
						Expect(err).To(MatchError("Incorrect usage. flag provided but not defined: -poooort\n\nUSAGE:\nbanana"))
						c := fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)
						Expect(c).To(Equal([]string{"help", "remove-access"}))

					})
				})
			})

			Context("when deleting the policies fails", func() {
				BeforeEach(func() {
					policyClient.DeletePoliciesReturns(errors.New("banana"))
				})
				It("wraps the error in a more helpful message", func() {
					_, err := runner.Remove()
					Expect(err).To(MatchError("deleting policies: banana"))
				})
			})

			Context("when getting the access token fails", func() {
				BeforeEach(func() {
					fakeCliConnection.AccessTokenReturns("", errors.New("banana"))
				})
				It("returns the error", func() {
					_, err := runner.Remove()
					Expect(err).To(MatchError("getting access token: banana"))
				})
			})

			Context("when getting the username fails", func() {
				BeforeEach(func() {
					fakeCliConnection.UsernameReturns("", errors.New("banana"))
				})
				It("returns an error", func() {
					_, err := runner.Remove()
					Expect(err).To(MatchError("could not resolve username: banana"))
				})
			})
		})

		Describe("Resolving App Names to Guids", func() {
			BeforeEach(func() {
				policyClient.GetAPIVersionReturns(1, nil)
			})

			Context("when there are errors talking to CC", func() {
				BeforeEach(func() {
					runner.Args = []string{"remove-access", "bad-access", "some-other-app", "--protocol", "tcp", "--port", "8888-9999"}
				})
				It("returns a useful error", func() {
					_, err := runner.Remove()
					Expect(err).To(MatchError("resolving source app: apple"))
				})
			})

			Context("when the source app could not be resolved to a GUID", func() {
				BeforeEach(func() {
					runner.Args = []string{"remove-access", "inaccessible-app", "some-other-app", "--protocol", "tcp", "--port", "8888-9999"}
				})
				It("returns a useful error", func() {
					_, err := runner.Remove()
					Expect(err).To(MatchError("resolving source app: inaccessible-app not found"))
				})
			})

			Context("when there are errors resolving destination app", func() {
				BeforeEach(func() {
					runner.Args = []string{"remove-access", "some-app", "not-some-other-app", "--protocol", "tcp", "--port", "8888-9999"}
				})
				It("returns a useful error", func() {
					_, err := runner.Remove()
					Expect(err).To(MatchError("resolving destination app: apple"))
				})
			})

			Context("when the destination app could not be resolved to a GUID", func() {
				BeforeEach(func() {
					runner.Args = []string{"remove-access", "some-app", "inaccessible-app", "--protocol", "tcp", "--port", "8888-9999"}
				})
				It("returns a useful error", func() {
					_, err := runner.Remove()
					Expect(err).To(MatchError("resolving destination app: inaccessible-app not found"))
				})
			})
		})

		Context("when using a single port", func() {
			BeforeEach(func() {
				policyClient.GetAPIVersionReturns(0, nil)
				runner.Args = []string{"remove-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999"}
			})

			Context("when the targeted environment is v1", func() {
				BeforeEach(func() {
					policyClient.GetAPIVersionReturns(1, nil)
					runner.Args = []string{"remove-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999"}
				})
				It("uses the v1 endpoint", func() {
					_, err := runner.Remove()
					Expect(err).NotTo(HaveOccurred())

					Expect(policyClient.DeletePoliciesCallCount()).To(Equal(1))
					Expect(policyClient.DeletePoliciesV0CallCount()).To(Equal(0))
				})
			})

			Context("when the policy is found", func() {
				It("removes the policy", func() {
					_, err := runner.Remove()
					Expect(err).NotTo(HaveOccurred())

					Expect(fakeCliConnection.GetAppCallCount()).To(Equal(2))
					Expect(fakeCliConnection.GetAppArgsForCall(0)).To(Equal("some-app"))
					Expect(fakeCliConnection.GetAppArgsForCall(1)).To(Equal("some-other-app"))

					Expect(policyClient.DeletePoliciesV0CallCount()).To(Equal(1))
					token, policies := policyClient.DeletePoliciesV0ArgsForCall(0)
					Expect(token).To(Equal("some-token"))
					Expect(policies).To(ConsistOf(api_v0.Policy{
						Source:      api_v0.Source{ID: "some-app-guid"},
						Destination: api_v0.Destination{ID: "some-other-app-guid", Port: 9999, Protocol: "tcp"}}))
				})
			})

			Context("when deleting the policies fails", func() {
				BeforeEach(func() {
					policyClient.DeletePoliciesV0Returns(errors.New("banana"))
				})
				It("wraps the error in a more helpful message", func() {
					_, err := runner.Remove()
					Expect(err).To(MatchError("deleting policies: banana"))
				})
			})

			Describe("Resolving App Names to Guids", func() {
				Context("when there are errors talking to CC", func() {
					BeforeEach(func() {
						runner.Args = []string{"remove-access", "bad-access", "some-other-app", "--protocol", "tcp", "--port", "9999"}
					})
					It("returns a useful error", func() {
						_, err := runner.Remove()
						Expect(err).To(MatchError("resolving source app: apple"))
					})
				})

				Context("when the source app could not be resolved to a GUID", func() {
					BeforeEach(func() {
						runner.Args = []string{"remove-access", "inaccessible-app", "some-other-app", "--protocol", "tcp", "--port", "9999"}
					})
					It("returns a useful error", func() {
						_, err := runner.Remove()
						Expect(err).To(MatchError("resolving source app: inaccessible-app not found"))
					})
				})

				Context("when there are errors resolving destination app", func() {
					BeforeEach(func() {
						runner.Args = []string{"remove-access", "some-app", "not-some-other-app", "--protocol", "tcp", "--port", "9999"}
					})
					It("returns a useful error", func() {
						_, err := runner.Remove()
						Expect(err).To(MatchError("resolving destination app: apple"))
					})
				})

				Context("when the destination app could not be resolved to a GUID", func() {
					BeforeEach(func() {
						runner.Args = []string{"remove-access", "some-app", "inaccessible-app", "--protocol", "tcp", "--port", "9999"}
					})
					It("returns a useful error", func() {
						_, err := runner.Remove()
						Expect(err).To(MatchError("resolving destination app: inaccessible-app not found"))
					})
				})
			})
		})

		Context("when we fail to determine an api version", func() {
			BeforeEach(func() {
				policyClient.GetAPIVersionReturns(0, errors.New("pineapple"))
				runner.Args = []string{"remove-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999"}
			})

			It("returns a useful error message", func() {
				_, err := runner.Remove()
				Expect(err).To(MatchError("could not determine networking API version: pineapple"))
			})
		})

		Context("when api version is too high", func() {
			BeforeEach(func() {
				policyClient.GetAPIVersionReturns(2, nil)
				runner.Args = []string{"remove-access", "some-app", "some-other-app", "--protocol", "tcp", "--port", "9999"}
			})

			It("returns a useful error message", func() {
				_, err := runner.Remove()
				Expect(err).To(MatchError("networking API version 2 not supported. consider using cf-cli native commands."))
			})
		})
	})
})
