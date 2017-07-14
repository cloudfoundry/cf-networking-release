package cli_plugin_test

import (
	"cli-plugin/cli_plugin"
	"cli-plugin/cli_plugin/fakes"
	"cli-plugin/styles"
	"errors"
	"log"

	libfakes "github.com/cloudfoundry-incubator/network-policy-client/src/policy_client/fakes"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/pluginfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin", func() {
	var (
		policyPlugin      cli_plugin.Plugin
		fakeCliConnection *pluginfakes.FakeCliConnection
		policyClient      *libfakes.ExternalPolicyClient
		versionGetter     *fakes.VersionGetter
	)

	BeforeEach(func() {
		policyClient = &libfakes.ExternalPolicyClient{}
		versionGetter = &fakes.VersionGetter{}
		policyPlugin = cli_plugin.Plugin{
			Styler:       styles.NewGroup(),
			Logger:       log.New(GinkgoWriter, "", 0),
			PolicyClient: policyClient,
			Version:      versionGetter,
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

	Describe("GetMetadata", func() {
		BeforeEach(func() {
			versionGetter.GetReturns(plugin.VersionType{
				Major: 4,
				Minor: 10,
				Build: 20,
			})
		})
		It("sets the version", func() {
			metadata := policyPlugin.GetMetadata()
			Expect(versionGetter.GetCallCount()).To(Equal(1))
			Expect(metadata.Version.Major).To(Equal(4))
			Expect(metadata.Version.Minor).To(Equal(10))
			Expect(metadata.Version.Build).To(Equal(20))
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
		})

		Context("when the port is out of range", func() {
			It("returns a useful error", func() {
				fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{"USAGE:", "banana"}, nil)
				_, err := cli_plugin.ValidateArgs(fakeCliConnection, []string{
					"command-arg", "some-app", "some-other-app", "--protocol", "tcp", "--port", "0",
				})
				Expect(err).To(MatchError("Incorrect usage. Port is not valid. Must be in range <1-65535>.\n\nUSAGE:\nbanana"))
				c := fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)
				Expect(c).To(Equal([]string{"help", "command-arg"}))
			})
		})

		Context("when the protocol is not tcp or udp", func() {
			It("returns a useful error", func() {
				fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{"USAGE:", "banana"}, nil)
				_, err := cli_plugin.ValidateArgs(fakeCliConnection, []string{
					"command-arg", "some-app", "some-other-app", "--protocol", "kiwi", "--port", "8080",
				})
				Expect(err).To(MatchError("Incorrect usage. Protocol is not valid. Must be tcp or udp.\n\nUSAGE:\nbanana"))
				c := fakeCliConnection.CliCommandWithoutTerminalOutputArgsForCall(0)
				Expect(c).To(Equal([]string{"help", "command-arg"}))
			})
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
