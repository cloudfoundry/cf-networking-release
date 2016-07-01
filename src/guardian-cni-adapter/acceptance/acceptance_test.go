package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type fakePluginLogData struct {
	Args  []string
	Env   map[string]string
	Stdin string
}

func getConfig(index int) string {
	return fmt.Sprintf(`
{
  "cniVersion": "0.1.0",
  "name": "some-net-%d",
  "type": "plugin-%d"
}`, index, index)
}

func getSkipConfig(index int) string {
	return fmt.Sprintf(`
{
  "cniVersion": "0.1.0",
  "name": "some-net-%d",
  "type": "plugin-%d",
  "skip_without_network": true
}`, index, index)
}

func expectedStdin(index int) string {
	return fmt.Sprintf(`
{
  "cniVersion": "0.1.0",
  "name": "some-net-%d",
  "type": "plugin-%d",
  "network": {
    "properties": {
      "some-key": "some-value",
      "some-other-key": "some-other-value"
    }
  }
}`, index, index)
}

func writeConfig(index int, outDir string) error {
	config := getConfig(index)
	outpath := filepath.Join(outDir, fmt.Sprintf("%d-plugin-%d.conf", 10*index, index))
	return ioutil.WriteFile(outpath, []byte(config), 0600)
}

func writeSkipConfig(index int, outDir string) error {
	config := getSkipConfig(index)
	outpath := filepath.Join(outDir, fmt.Sprintf("%d-plugin-%d.conf", 10*index, index))
	return ioutil.WriteFile(outpath, []byte(config), 0600)
}

func sameFile(path1, path2 string) bool {
	fi1, err := os.Stat(path1)
	Expect(err).NotTo(HaveOccurred())

	fi2, err := os.Stat(path2)
	Expect(err).NotTo(HaveOccurred())
	return os.SameFile(fi1, fi2)
}

const DEFAULT_TIMEOUT = "10s"

var _ = Describe("Guardian CNI adapter", func() {
	var (
		cniConfigDir       string
		fakePid            int
		fakeLogDir         string
		expectedNetNSPath  string
		bindMountRoot      string
		containerHandle    string
		fakeProcess        *os.Process
		fakeConfigFilePath string
		adapterLogFilePath string
	)

	BeforeEach(func() {
		var err error
		cniConfigDir, err = ioutil.TempDir("", "cni-config-")
		Expect(err).NotTo(HaveOccurred())

		fakeLogDir, err = ioutil.TempDir("", "fake-logs-")
		Expect(err).NotTo(HaveOccurred())

		containerHandle = "some-container-handle"

		sleepCmd := exec.Command("/bin/sleep", "1000")
		Expect(sleepCmd.Start()).To(Succeed())
		fakeProcess = sleepCmd.Process

		fakePid = fakeProcess.Pid

		bindMountRoot, err = ioutil.TempDir("", "bind-mount-root")
		Expect(err).NotTo(HaveOccurred())

		expectedNetNSPath = fmt.Sprintf("%s/%s", bindMountRoot, containerHandle)

		adapterLogDir, err := ioutil.TempDir("", "adapter-log-dir")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.RemoveAll(adapterLogDir)).To(Succeed()) // directory need not exist
		adapterLogFilePath = filepath.Join(adapterLogDir, "some-container-handle.log")

		Expect(writeConfig(0, cniConfigDir)).To(Succeed())
		Expect(writeConfig(1, cniConfigDir)).To(Succeed())
		Expect(writeConfig(2, cniConfigDir)).To(Succeed())

		configFile, err := ioutil.TempFile("", "adapter-config-")
		Expect(err).NotTo(HaveOccurred())
		fakeConfigFilePath = configFile.Name()
		config := map[string]string{
			"cni_plugin_dir": cniPluginDir,
			"cni_config_dir": cniConfigDir,
			"bind_mount_dir": bindMountRoot,
			"log_dir":        adapterLogDir,
		}
		configBytes, err := json.Marshal(config)
		Expect(err).NotTo(HaveOccurred())
		_, err = configFile.Write(configBytes)
		Expect(err).NotTo(HaveOccurred())
		Expect(configFile.Close()).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.Remove(fakeConfigFilePath)).To(Succeed())
		Expect(os.RemoveAll(cniConfigDir)).To(Succeed())
		Expect(os.RemoveAll(fakeLogDir)).To(Succeed())
		Expect(fakeProcess.Kill()).To(Succeed())
	})

	Describe("CNI plugin lifecycle events", func() {
		var upCommand, downCommand *exec.Cmd

		BeforeEach(func() {
			upCommand = exec.Command(pathToAdapter)
			upCommand.Env = []string{"FAKE_LOG_DIR=" + fakeLogDir}
			upCommand.Stdin = strings.NewReader(fmt.Sprintf(`{ "pid": %d }`, fakePid))
			upCommand.Args = []string{
				pathToAdapter,
				"--configFile", fakeConfigFilePath,
				"--action", "up",
				"--handle", "some-container-handle",
				"--network", "garden-network-spec",
			}

			downCommand = exec.Command(pathToAdapter)
			downCommand.Env = []string{"FAKE_LOG_DIR=" + fakeLogDir}
			downCommand.Stdin = strings.NewReader(`{}`)
			downCommand.Args = []string{
				pathToAdapter,
				"--action", "down",
				"--handle", "some-container-handle",
				"--configFile", fakeConfigFilePath,
				"--network", "garden-network-spec",
			}
		})

		Context("when network-properties are present", func() {
			BeforeEach(func() {
				upCommand.Args = append(
					upCommand.Args,
					"--properties", `{ "some-key": "some-value", "some-other-key": "some-other-value" }`,
				)

				downCommand.Args = append(
					downCommand.Args,
					"--properties", `{ "some-key": "some-value", "some-other-key": "some-other-value" }`,
				)
			})

			It("should call CNI ADD and DEL", func() {
				By("calling up")
				upSession, err := gexec.Start(upCommand, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(upSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))

				By("checking that every CNI plugin in the plugin directory got called with ADD")
				for i := 0; i < 3; i++ {
					logFileContents, err := ioutil.ReadFile(filepath.Join(fakeLogDir, fmt.Sprintf("plugin-%d.log", i)))
					Expect(err).NotTo(HaveOccurred())
					var pluginCallInfo fakePluginLogData
					Expect(json.Unmarshal(logFileContents, &pluginCallInfo)).To(Succeed())

					Expect(pluginCallInfo.Stdin).To(MatchJSON(expectedStdin(i)))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_COMMAND", "ADD"))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_CONTAINERID", containerHandle))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_IFNAME", fmt.Sprintf("eth%d", i)))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_PATH", cniPluginDir))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_NETNS", expectedNetNSPath))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_ARGS", ""))
				}

				By("checking that the fake process's network namespace has been bind-mounted into the filesystem")
				Expect(sameFile(expectedNetNSPath, fmt.Sprintf("/proc/%d/ns/net", fakePid))).To(BeTrue())

				By("calling down")
				downSession, err := gexec.Start(downCommand, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(downSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))

				By("checking that every CNI plugin in the plugin directory got called with DEL")
				for i := 0; i < 3; i++ {
					logFileContents, err := ioutil.ReadFile(filepath.Join(fakeLogDir, fmt.Sprintf("plugin-%d.log", i)))
					Expect(err).NotTo(HaveOccurred())
					var pluginCallInfo fakePluginLogData
					Expect(json.Unmarshal(logFileContents, &pluginCallInfo)).To(Succeed())

					Expect(pluginCallInfo.Stdin).To(MatchJSON(getConfig(i)))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_COMMAND", "DEL"))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_CONTAINERID", containerHandle))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_IFNAME", fmt.Sprintf("eth%d", i)))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_PATH", cniPluginDir))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_NETNS", expectedNetNSPath))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_ARGS", ""))
				}

				By("checking that the bind-mounted namespace has been removed")
				Expect(expectedNetNSPath).NotTo(BeAnExistingFile())
			})
		})

		Context("when the network spec is missing", func() {
			BeforeEach(func() {
				Expect(writeSkipConfig(0, cniConfigDir)).To(Succeed())
				Expect(writeConfig(1, cniConfigDir)).To(Succeed())
				Expect(writeConfig(2, cniConfigDir)).To(Succeed())
				Expect(writeSkipConfig(3, cniConfigDir)).To(Succeed())

				upCommand.Args = append(
					upCommand.Args,
					"--properties", `{}`,
				)

				downCommand.Args = append(
					downCommand.Args,
					"--properties", `{}`,
				)
			})

			It("calls CNI plugins for which skip_without_network is false", func() {
				upSession, err := gexec.Start(upCommand, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(upSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))

				By("checking plugins with skip_without_network were not called")
				Expect(filepath.Join(fakeLogDir, "plugin-0.log")).NotTo(BeAnExistingFile())
				Expect(filepath.Join(fakeLogDir, "plugin-3.log")).NotTo(BeAnExistingFile())

				By("checking that other CNI plugins got called with ADD")
				for i := 1; i < 3; i++ {
					logFileContents, err := ioutil.ReadFile(filepath.Join(fakeLogDir, fmt.Sprintf("plugin-%d.log", i)))
					Expect(err).NotTo(HaveOccurred())
					var pluginCallInfo fakePluginLogData
					Expect(json.Unmarshal(logFileContents, &pluginCallInfo)).To(Succeed())

					Expect(pluginCallInfo.Stdin).To(MatchJSON(getConfig(i)))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_COMMAND", "ADD"))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_CONTAINERID", containerHandle))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_IFNAME", fmt.Sprintf("eth%d", i)))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_PATH", cniPluginDir))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_NETNS", expectedNetNSPath))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_ARGS", ""))
				}

				By("logging the plugin output / result from up")
				logContents, err := ioutil.ReadFile(adapterLogFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(logContents).To(ContainSubstring("up result for name=some-net-1, type=plugin-1: "))
				Expect(logContents).To(ContainSubstring("up result for name=some-net-2, type=plugin-2: "))
				Expect(logContents).To(ContainSubstring("169.254.1.2"))

				By("checking that the fake process's network namespace has been bind-mounted into the filesystem")
				Expect(sameFile(expectedNetNSPath, fmt.Sprintf("/proc/%d/ns/net", fakePid))).To(BeTrue())

				By("calling down")
				downSession, err := gexec.Start(downCommand, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(downSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))

				By("checking plugins with skip_without_network were not called")
				Expect(filepath.Join(fakeLogDir, "plugin-0.log")).NotTo(BeAnExistingFile())
				Expect(filepath.Join(fakeLogDir, "plugin-3.log")).NotTo(BeAnExistingFile())

				By("checking that other CNI plugins got called with DEL")
				for i := 1; i < 3; i++ {
					logFileContents, err := ioutil.ReadFile(filepath.Join(fakeLogDir, fmt.Sprintf("plugin-%d.log", i)))
					Expect(err).NotTo(HaveOccurred())
					var pluginCallInfo fakePluginLogData
					Expect(json.Unmarshal(logFileContents, &pluginCallInfo)).To(Succeed())

					Expect(pluginCallInfo.Stdin).To(MatchJSON(getConfig(i)))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_COMMAND", "DEL"))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_CONTAINERID", containerHandle))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_IFNAME", fmt.Sprintf("eth%d", i)))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_PATH", cniPluginDir))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_NETNS", expectedNetNSPath))
					Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_ARGS", ""))
				}

				By("logging the plugin output / result from down")
				logContents, err = ioutil.ReadFile(adapterLogFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(logContents).To(ContainSubstring("down complete for name=some-net-1, type=plugin-1"))
				Expect(logContents).To(ContainSubstring("down complete for name=some-net-2, type=plugin-2"))

				By("checking that the bind-mounted namespace has been removed")
				Expect(expectedNetNSPath).NotTo(BeAnExistingFile())
			})
		})
	})
})
