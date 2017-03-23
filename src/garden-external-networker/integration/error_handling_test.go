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
	"github.com/onsi/gomega/gbytes"
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
			"cni_plugin_dir": dir,
			"cni_config_dir": dir,
			"bind_mount_dir": dir,
			"state_file":     stateFilePath.Name(),
			"start_port":     1234,
			"total_ports":    56,
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
		Context("when there's a generic error in main", func() {
			It("prints the error to stderr with the lager logger", func() {
				command.Args = []string{
					paths.PathToAdapter,
					"invalidArg",
				}
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say(".*timestamp.*source.*container-networking.garden-external-networker.*message.*container-networking.garden-external-networker.error.*log_level.*2.*data.*error.*parse args: unexpected extra args:.*invalidArg.*}.*}"))
			})
		})
		Context("when stdin is not valid JSON", func() {
			It("should exit status 1 and print an error to stderr", func() {
				command.Stdin = strings.NewReader("{{{bad")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				By("checking that the error was logged to stderr")
				Expect(session.Err.Contents()).To(ContainSubstring("invalid character"))
			})
		})

		Context("when the stdin JSON is missing a pid field", func() {
			It("should exit status 1 and print an error to stderr", func() {
				command.Stdin = strings.NewReader(`{ "something": 12 }`)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session, "2s").Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				Expect(session.Err.Contents()).To(ContainSubstring("missing pid"))
			})
		})

		Context("when the provided pid is not an integer", func() {
			It("should exit status 1 and print an error to stderr", func() {
				command.Stdin = strings.NewReader(`{ "pid": "not-a-number"  }`)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				Expect(session.Err.Contents()).To(ContainSubstring(`cannot unmarshal string into Go value of type int`))
			})
		})

		Context("when the action is incorrect", func() {
			It("should return an error", func() {
				command.Args[1] = "--action=some-invalid-action"

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				Expect(session.Err.Contents()).To(ContainSubstring(`unrecognized action: some-invalid-action`))
			})
		})

		Context("when an unknown flag is provided", func() {
			It("should return an error", func() {
				command.Args = append(command.Args, "--banana")

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				Expect(session.Err.Contents()).To(ContainSubstring(`flag provided but not defined: -banana`))
			})
		})

		Context("when an unknown positional arg is provided", func() {
			It("should return an error", func() {
				command.Args = append(command.Args, "something-else")

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				Expect(session.Err.Contents()).To(ContainSubstring(`unexpected extra args: [something-else]`))
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
			func(missingFlag string) {
				command.Args = removeArrayElement(command.Args, "--"+missingFlag)

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				By("checking that process exits with an err")
				Eventually(session).Should(gexec.Exit(1))

				By("checking that the error was logged to stderr")
				Expect(session.Out.Contents()).To(BeEmpty())
				expectedErrorString := fmt.Sprintf("missing required flag '%s'", missingFlag)
				Expect(session.Err.Contents()).To(ContainSubstring(expectedErrorString))
			},
			Entry("action", "action"),
			Entry("handle", "handle"),
			Entry("configFile", "configFile"),
		)

		DescribeTable("missing required config",
			func(missingKey string) {
				delete(defaultConfig, missingKey)
				writeConfig(defaultConfig)

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				By("checking that process exits with an err")
				Eventually(session).Should(gexec.Exit(1))

				By("checking that the error was logged to stderr")
				Expect(session.Out.Contents()).To(BeEmpty())
				expectedErrorString := fmt.Sprintf("missing required config '%s'", missingKey)
				Expect(session.Err.Contents()).To(ContainSubstring(expectedErrorString))
			},
			Entry("cni_plugin_dir", "cni_plugin_dir"),
			Entry("cni_config_dir", "cni_config_dir"),
			Entry("bind_mount_dir", "bind_mount_dir"),
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
					Expect(session.Err.Contents()).To(ContainSubstring(`this is a plugin for Garden-runC.  Don't run it directly.`))
				},
				Entry("no args", []string{paths.PathToAdapter}),
				Entry("short help", []string{paths.PathToAdapter, "-h"}),
				Entry("long help", []string{paths.PathToAdapter, "--help"}),
			)
		})
	})
})
