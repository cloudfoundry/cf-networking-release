package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/coreos/go-iptables/iptables"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type fakePluginLogData struct {
	Args  []string
	Env   map[string]string
	Stdin string
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
      "app_id": "some-group-id"
    }
  }
}`, index, index)
}

func writeConfig(index int, outDir string) error {
	config := fmt.Sprintf(`
{
  "cniVersion": "0.1.0",
  "name": "some-net-%d",
  "type": "plugin-%d"
}`, index, index)
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

var netmanAgentReceivedData = ``
var netmanAgentReceivedMethod = ``

const DEFAULT_TIMEOUT = "10s"

var _ = Describe("Garden External Networker", func() {
	var (
		cniConfigDir           string
		fakePid                int
		fakeLogDir             string
		expectedNetNSPath      string
		bindMountRoot          string
		stateFilePath          string
		containerHandle        string
		fakeProcess            *os.Process
		fakeConfigFilePath     string
		adapterLogFilePath     string
		upCommand, downCommand *exec.Cmd
		netOutCommand          *exec.Cmd
		adapterLogDir          string
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

		stateFile, err := ioutil.TempFile("", "external-networker-state.json")
		Expect(err).NotTo(HaveOccurred())
		Expect(stateFile.Close()).To(Succeed())
		stateFilePath = stateFile.Name()

		adapterLogDir, err = ioutil.TempDir("", "adapter-log-dir")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.RemoveAll(adapterLogDir)).To(Succeed()) // directory need not exist
		adapterLogFilePath = filepath.Join(adapterLogDir, "some-container-handle.log")

		Expect(writeConfig(0, cniConfigDir)).To(Succeed())
		Expect(writeConfig(1, cniConfigDir)).To(Succeed())
		Expect(writeConfig(2, cniConfigDir)).To(Succeed())

		netmanAgentReceivedData = ""
		netmanAgentReceivedMethod = ""

		configFile, err := ioutil.TempFile("", "adapter-config-")
		Expect(err).NotTo(HaveOccurred())
		fakeConfigFilePath = configFile.Name()
		config := map[string]interface{}{
			"cni_plugin_dir":  cniPluginDir,
			"cni_config_dir":  cniConfigDir,
			"bind_mount_dir":  bindMountRoot,
			"overlay_network": "10.255.0.0/16",
			"state_file":      stateFilePath,
			"start_port":      60000,
			"total_ports":     56,
		}
		configBytes, err := json.Marshal(config)
		Expect(err).NotTo(HaveOccurred())
		_, err = configFile.Write(configBytes)
		Expect(err).NotTo(HaveOccurred())
		Expect(configFile.Close()).To(Succeed())

		upCommand = exec.Command(pathToAdapter)
		upCommand.Env = append(os.Environ(), "FAKE_LOG_DIR="+fakeLogDir)
		upCommand.Stdin = strings.NewReader(fmt.Sprintf(`{ "pid": %d }`, fakePid))
		upCommand.Args = []string{
			pathToAdapter,
			"--configFile", fakeConfigFilePath,
			"--action", "up",
			"--handle", "some-container-handle",
		}
		upCommand.Args = append(
			upCommand.Args,
			"--properties", `{ "some-key": "some-value", "app_id": "some-group-id" }`,
		)

		downCommand = exec.Command(pathToAdapter)
		downCommand.Env = append(os.Environ(), "FAKE_LOG_DIR="+fakeLogDir)
		downCommand.Stdin = strings.NewReader(`{}`)
		downCommand.Args = []string{
			pathToAdapter,
			"--action", "down",
			"--handle", "some-container-handle",
			"--configFile", fakeConfigFilePath,
		}
		downCommand.Args = append(
			downCommand.Args,
			"--properties", `{ "some-key": "some-value", "app_id": "some-group-id" }`,
		)

		netOutCommand = exec.Command(pathToAdapter)
		netOutCommand.Env = append(os.Environ(), "FAKE_LOG_DIR="+fakeLogDir)
		netOutCommand.Stdin = strings.NewReader(`{}`)
		netOutCommand.Args = []string{
			pathToAdapter,
			"--action", "net-out",
			"--handle", "some-container-handle",
			"--configFile", fakeConfigFilePath,
			"--properties", `{ "container_ip":"169.254.1.2","netout_rule":{"protocol": 1, "networks": [{"start":"1.1.1.1","end":"2.2.2.2"}], "ports": [{"start":9000,"end":9999}]}}`,
		}
	})

	AfterEach(func() {
		Expect(os.Remove(fakeConfigFilePath)).To(Succeed())
		Expect(os.RemoveAll(cniConfigDir)).To(Succeed())
		Expect(os.RemoveAll(fakeLogDir)).To(Succeed())
		Expect(fakeProcess.Kill()).To(Succeed())

		ipt, err := iptables.New()
		Expect(err).NotTo(HaveOccurred())
		Expect(ipt.ClearChain("filter", "netout--some-container-handl")).To(Succeed())
		Expect(ipt.ClearChain("filter", "FORWARD")).To(Succeed())
		Expect(ipt.DeleteChain("filter", "netout--some-container-handl")).To(Succeed())
	})

	It("should call CNI ADD and DEL", func() {
		By("calling up")
		upSession, err := gexec.Start(upCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(upSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
		Expect(upSession.Out.Contents()).To(MatchJSON(`{ "properties": {"garden.network.container-ip": "169.254.1.2",  "garden.network.host-ip": "255.255.255.255"} }`))

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

			Expect(pluginCallInfo.Stdin).To(MatchJSON(expectedStdin(i)))
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

	It("writes NetOut rules", func() {
		By("calling up")
		upSession, err := gexec.Start(upCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(upSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
		Expect(upSession.Out.Contents()).To(MatchJSON(`{ "properties": {"garden.network.container-ip": "169.254.1.2",  "garden.network.host-ip": "255.255.255.255"} }`))

		By("checking that the default rules are created for that container")
		iptablesCommand := exec.Command("iptables", "-t", "filter", "-S")
		iptSession, err := gexec.Start(iptablesCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(iptSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
		Expect(iptSession.Out.Contents()).To(ContainSubstring(`netout--some-container-handl -s 169.254.1.2/32 ! -d 10.255.0.0/16 -m state --state RELATED,ESTABLISHED -j RETURN`))
		Expect(iptSession.Out.Contents()).To(ContainSubstring(`netout--some-container-handl -s 169.254.1.2/32 ! -d 10.255.0.0/16 -j REJECT --reject-with icmp-port-unreachable`))

		By("calling netout")
		netOutSession, err := gexec.Start(netOutCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(netOutSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
		iptablesCommand = exec.Command("iptables", "-t", "filter", "-S")
		iptSession, err = gexec.Start(iptablesCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(iptSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
		Expect(iptSession.Out.Contents()).To(ContainSubstring(`netout--some-container-handl -s 169.254.1.2/32 -p tcp -m iprange --dst-range 1.1.1.1-2.2.2.2 -m tcp --dport 9000:9999 -j RETURN`))

		By("calling netout again but without ports or protocols")
		netOutCommand = exec.Command(pathToAdapter)
		netOutCommand.Env = append(os.Environ(), "FAKE_LOG_DIR="+fakeLogDir)
		netOutCommand.Stdin = strings.NewReader(`{}`)
		netOutCommand.Args = []string{
			pathToAdapter,
			"--action", "net-out",
			"--handle", "some-container-handle",
			"--configFile", fakeConfigFilePath,
			"--properties", `{ "container_ip":"169.254.1.2","netout_rule":{"networks": [{"start":"3.3.3.3","end":"4.4.4.4"}]}}`,
		}
		netOutSession, err = gexec.Start(netOutCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(netOutSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
		iptablesCommand = exec.Command("iptables", "-t", "filter", "-S")
		iptSession, err = gexec.Start(iptablesCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(iptSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
		Expect(iptSession.Out.Contents()).To(ContainSubstring(`netout--some-container-handl -s 169.254.1.2/32 -p tcp -m iprange --dst-range 1.1.1.1-2.2.2.2 -m tcp --dport 9000:9999 -j RETURN`))
		Expect(iptSession.Out.Contents()).To(ContainSubstring(`netout--some-container-handl -s 169.254.1.2/32 -m iprange --dst-range 3.3.3.3-4.4.4.4 -j RETURN`))
	})

	Describe("NetIn", func() {
		var netInCommand *exec.Cmd

		BeforeEach(func() {
			netInCommand = exec.Command(pathToAdapter)
			netInCommand.Env = append(os.Environ(), "FAKE_LOG_DIR="+fakeLogDir)
			netInCommand.Stdin = strings.NewReader(`{}`)
			netInCommand.Args = []string{
				pathToAdapter,
				"--action", "net-in",
				"--handle", "some-container-handle",
				"--configFile", fakeConfigFilePath,
			}
			netInCommand.Args = append(
				netInCommand.Args,
				"--properties", `{ "host-ip": "1.2.3.4", "host-port": "0", "container-ip": "10.0.0.2", "container-port": "8080", "app_id": "some-group-id" }`,
			)
		})

		It("writes iptables rules for NetIn", func() {
			By("ensuring iptables chain is present for the container handle")
			iptablesCmd := exec.Command("iptables", "-w", "-t", "nat", "-N", "netin--some-container-handle")
			iptablesSession, err := gexec.Start(iptablesCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(iptablesSession).Should(gexec.Exit(0))

			By("returning the ports allocated")
			netInSession, err := gexec.Start(netInCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(netInSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
			var result struct {
				HostPort      int `json:"host_port"`
				ContainerPort int `json:"container_port"`
			}
			Expect(json.Unmarshal(netInSession.Out.Contents(), &result)).To(Succeed())
			Expect(result.HostPort).To(BeNumerically(">=", 60000))
			Expect(result.ContainerPort).To(Equal(8080))

			By("calling out to iptables")
			iptablesCmd = exec.Command("iptables", "-w", "-S", "-t", "nat")
			iptablesSession, err = gexec.Start(iptablesCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(iptablesSession).Should(gexec.Exit(0))
			allRules := strings.Split(string(iptablesSession.Out.Contents()), "\n")
			expectedRule := `-A netin--some-container-handle -d 1.2.3.4/32 -p tcp -m tcp --dport 60000 -m comment --comment "dst:some-group-id" -j DNAT --to-destination 10.0.0.2:8080`
			Expect(allRules).To(ContainElement(expectedRule))

			By("seeing that the allocated port is stored to the state file on disk")
			stateFileBytes, err := ioutil.ReadFile(stateFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(stateFileBytes).To(ContainSubstring(fmt.Sprintf("%d", result.HostPort)))
		})
	})
})
