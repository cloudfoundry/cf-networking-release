package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Guardian CNI adapter", func() {
	var (
		command            *exec.Cmd
		cniConfigDir       string
		fakePid            int
		fakeLogDir         string
		adapterLogFilePath string
		fakeConfigFilePath string
		defaultConfig      map[string]string
	)

	var writeConfig = func(configHash map[string]string) {
		configBytes, err := json.Marshal(configHash)
		Expect(err).NotTo(HaveOccurred())
		err = ioutil.WriteFile(fakeConfigFilePath, configBytes, 0600)
		Expect(err).NotTo(HaveOccurred())
	}

	BeforeEach(func() {
		var err error
		adapterLogDir, err := ioutil.TempDir("", "adapter-log-dir")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.RemoveAll(adapterLogDir)).To(Succeed()) // directory need not exist

		adapterLogFilePath = filepath.Join(adapterLogDir, "some-container-handle.log")

		configFile, err := ioutil.TempFile("", "adapter-config-")
		Expect(err).NotTo(HaveOccurred())
		Expect(configFile.Close()).To(Succeed())

		fakeConfigFilePath = configFile.Name()
		defaultConfig = map[string]string{
			"cni_plugin_dir": "/some/cni/plugin/dir",
			"cni_config_dir": "/some/cni/config/dir",
			"bind_mount_dir": "/some/bind/mount/dir",
			"log_dir":        adapterLogDir,
		}
		writeConfig(defaultConfig)

		command = exec.Command(pathToAdapter)
		command.Args = []string{pathToAdapter,
			"--action=up",
			"--handle=some-container-handle",
			"--properties=some-network-spec",
			"--configFile=" + fakeConfigFilePath,
		}

		fakePid = rand.Intn(30000)
		command.Stdin = strings.NewReader(fmt.Sprintf(`{ "pid": %d }`, fakePid))
	})

	AfterEach(func() {
		Expect(os.RemoveAll(cniConfigDir)).To(Succeed())
		Expect(os.RemoveAll(fakeLogDir)).To(Succeed())
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
				Expect(session.Err.Contents()).To(ContainSubstring("json"))
				Expect(session.Err.Contents()).To(ContainSubstring("{{{bad"))

			})
		})

		Context("when the stdin JSON is missing a pid field", func() {
			It("should exit status 1 and print an error to stderr", func() {
				command.Stdin = strings.NewReader(`{ "something": 12 }`)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
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
				Expect(session.Err.Contents()).To(ContainSubstring(`action: some-invalid-action is unrecognized`))

				By("checking that the error was logged to a file")
				Expect(ioutil.ReadFile(adapterLogFilePath)).To(ContainSubstring("action: some-invalid-action"))
			})

			Context("when the log file already exists", func() {
				It("should append to it", func() {
					Expect(os.MkdirAll(filepath.Dir(adapterLogFilePath), 0644)).To(Succeed())
					Expect(ioutil.WriteFile(adapterLogFilePath, []byte("some existing logs\n"), 0644)).To(Succeed())

					command.Args[1] = "--action=some-invalid-action"
					session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())

					Eventually(session).Should(gexec.Exit(1))
					Expect(ioutil.ReadFile(adapterLogFilePath)).To(HavePrefix("some existing logs"))
					Expect(ioutil.ReadFile(adapterLogFilePath)).To(ContainSubstring("action: some-invalid-action"))
				})
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

				By("checking that the error was logged to a file")
				if missingFlag != "handle" && missingFlag != "configFile" {
					Expect(ioutil.ReadFile(adapterLogFilePath)).To(ContainSubstring(expectedErrorString))
				}
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
					Expect(session.Err.Contents()).To(ContainSubstring(`this is a OCI prestart/poststop hook.  see https://github.com/opencontainers/specs/blob/master/runtime-config.md`))
				},
				Entry("no args", []string{pathToAdapter}),
				Entry("short help", []string{pathToAdapter, "-h"}),
				Entry("long help", []string{pathToAdapter, "--help"}),
			)
		})
	})
})
