package integration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Garden External Networker errors", func() {
	var (
		command            *exec.Cmd
		fakeConfigFilePath string
		defaultConfig      map[string]interface{}
	)

	var writeConfig = func(configHash map[string]interface{}) {
		configBytes, err := json.Marshal(configHash)
		Expect(err).NotTo(HaveOccurred())
		err = ioutil.WriteFile(fakeConfigFilePath, configBytes, 0600)
		Expect(err).NotTo(HaveOccurred())
	}

	BeforeEach(func() {
		var err error
		configFile, err := ioutil.TempFile("", "adapter-config-")
		Expect(err).NotTo(HaveOccurred())
		Expect(configFile.Close()).To(Succeed())

		dir, err := ioutil.TempDir("", "fake-cni-dir")
		Expect(err).ToNot(HaveOccurred())

		stateFilePath, err := ioutil.TempFile("", "external-networker-state.json")
		Expect(err).NotTo(HaveOccurred())

		fakeConfigFilePath = configFile.Name()
		defaultConfig = map[string]interface{}{
			"cni_plugin_dir":      dir,
			"cni_config_dir":      dir,
			"bind_mount_dir":      dir,
			"state_file":          stateFilePath.Name(),
			"start_port":          1234,
			"total_ports":         56,
			"log_prefix":          "prefix",
			"iptables_lock_file":  GlobalIPTablesLockFile,
			"proxy_redirect_cidr": "",
			"proxy_port":          9999,
			"proxy_uid":           42,
		}
		writeConfig(defaultConfig)

		command = exec.Command(paths.PathToAdapter)
		command.Args = []string{paths.PathToAdapter,
			"--action=up",
			"--handle=some-container-handle",
			"--configFile=" + fakeConfigFilePath,
		}
		command.Env = []string{"PATH=/sbin"}

		command.Stdin = strings.NewReader(fmt.Sprintf(`{ "pid": %d }`, GinkgoParallelNode()))
	})

	Context("when inputs are invalid", func() {
		Context("when stdin is not valid JSON", func() {
			It("should exit status 1 and print an error to stderr", func() {
				command.Stdin = strings.NewReader("{{{bad")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				By("checking that the error was logged to stderr")
				Expect(string(session.Err.Contents())).To(ContainSubstring("prefix: invalid character"))
			})
		})

		Context("when the provided pid is not an integer", func() {
			It("should exit status 1 and print an error to stderr", func() {
				command.Stdin = strings.NewReader(`{ "pid": "not-a-number"  }`)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				Expect(string(session.Err.Contents())).To(MatchRegexp(`prefix: json: cannot unmarshal string into Go.*type int`))
			})
		})

		Context("when the action is incorrect", func() {
			It("should return an error", func() {
				command.Args[1] = "--action=some-invalid-action"

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				Expect(string(session.Err.Contents())).To(ContainSubstring(`prefix: unrecognized action: some-invalid-action`))
			})
		})

		Context("when neither a valid pid or fd3 are provided", func() {
			It("should exit status 1 and print an error to stderr", func() {
				command.Stdin = strings.NewReader(`{ "pid": 0 }`)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				Expect(session.Err.Contents()).NotTo(BeEmpty())
			})
		})

		Context("when an unknown flag is provided", func() {
			It("should return an error", func() {
				command.Args = append(command.Args, "--banana")

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				Expect(string(session.Err.Contents())).To(ContainSubstring(`cfnetworking: parse args: flag provided but not defined: -banana`))
			})
		})

		Context("when an unknown positional arg is provided", func() {
			It("should return an error", func() {
				command.Args = append(command.Args, "something-else")

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				Expect(session.Err.Contents()).To(ContainSubstring(`prefix: parse args: unexpected extra args: [something-else]`))
			})
		})

		var removeArrayElement = func(src []string, elementToRemove string) []string {
			reduced := []string{}
			for _, element := range src {
				if !strings.HasPrefix(element, elementToRemove) {
					reduced = append(reduced, element)
				}
			}

			return reduced
		}

		DescribeTable("missing required arguments",
			func(missingFlag, prefix string) {
				command.Args = removeArrayElement(command.Args, "--"+missingFlag)

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				By("checking that process exits with an err")
				Eventually(session).Should(gexec.Exit(1))

				By("checking that the error was logged to stderr")
				Expect(session.Out.Contents()).To(BeEmpty())
				expectedErrorString := fmt.Sprintf("%s: parse args: missing required flag '%s'", prefix, missingFlag)
				Expect(string(session.Err.Contents())).To(ContainSubstring(expectedErrorString))
			},
			Entry("action", "action", "prefix"),
			Entry("handle", "handle", "prefix"),
			Entry("configFile", "configFile", "cfnetworking"),
		)

		DescribeTable("missing required config",
			func(missingKey, prefix string) {
				delete(defaultConfig, missingKey)
				writeConfig(defaultConfig)

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				By("checking that process exits with an err")
				Eventually(session).Should(gexec.Exit(1))

				By("checking that the error was logged to stderr")
				Expect(session.Out.Contents()).To(BeEmpty())
				expectedErrorString := fmt.Sprintf("%s: parse args: missing required config '%s'", prefix, missingKey)
				Expect(string(session.Err.Contents())).To(ContainSubstring(expectedErrorString))
			},
			Entry("cni_plugin_dir", "cni_plugin_dir", "prefix"),
			Entry("cni_config_dir", "cni_config_dir", "prefix"),
			Entry("bind_mount_dir", "bind_mount_dir", "prefix"),
			Entry("log_prefix", "log_prefix", "cfnetworking"),
		)

		Context("when the user doesn't know what to do", func() {
			DescribeTable("arguments that indicate ignorance",
				func(args []string) {
					command.Args = args
					command.Stdin = strings.NewReader("invalid json")

					session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())

					Eventually(session).Should(gexec.Exit(1))
					Expect(session.Out.Contents()).To(BeEmpty())
					Expect(session.Err.Contents()).To(ContainSubstring(`cfnetworking: this is a plugin for Garden-runC.  Don't run it directly.`))
				},
				Entry("no args", []string{paths.PathToAdapter}),
				Entry("short help", []string{paths.PathToAdapter, "-h"}),
				Entry("long help", []string{paths.PathToAdapter, "--help"}),
			)
		})
	})
})
