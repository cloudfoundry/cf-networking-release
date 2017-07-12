package integration_test

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"lib/datastore"
	"lib/filelock"
	"lib/serial"
	"log-transformer/config"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var (
	outputDir  string
	outputFile string
)

var _ = Describe("Integration", func() {
	var (
		session               *gexec.Session
		conf                  config.LogTransformer
		kernelLogFile         *os.File
		containerMetadataFile *os.File
		store                 *datastore.Store
		configFilePath        string
	)

	EGRESS_DENIED_JSON := `{
			"timestamp": "some-timestamp",
			"source": "cfnetworking.iptables",
			"message": "cfnetworking.iptables.egress-denied",
			"log_level": 1,
			"data": {
				"source": {
					"container_id": "container-handle-1-longer-than-29-chars",
					"app_guid": "app_id_1",
					"space_guid": "space_id_1",
					"organization_guid": "organization_id_1"
				},
				"packet": {
					"direction": "egress",
					"allowed": false,
					"src_ip": "10.255.0.1",
					"src_port":45564,
					"dst_ip": "10.10.10.10",
					"dst_port": 25555,
					"protocol": "UDP",
					"mark": "0x1",
					"icmp_code": 0,
					"icmp_type": 0
				}
			}
		}`
	EGRESS_ALLOWED_JSON := `{
			"timestamp": "some-timestamp",
			"source": "cfnetworking.iptables",
			"message": "cfnetworking.iptables.egress-allowed",
			"log_level": 1,
			"data": {
				"source": {
					"container_id": "container-handle-1-longer-than-29-chars",
					"app_guid": "app_id_1",
					"space_guid": "space_id_1",
					"organization_guid": "organization_id_1"
				},
				"packet": {
					"direction": "egress",
					"allowed": true,
					"src_ip": "10.255.0.1",
					"src_port": 36556,
					"dst_ip": "10.10.10.10",
					"dst_port": 11111,
					"protocol": "UDP",
					"mark": "0x1",
					"icmp_code": 0,
					"icmp_type": 0
				}
			}
		}`

	EGRESS_ALLOWD_KERNEL_LOG := "Jun 28 18:21:24 localhost kernel: [100471.222018] OK_container-handle-1-longer IN=s-010255178004 OUT=eth0 MAC=aa:aa:0a:ff:b2:04:ee:ee:0a:ff:b2:04:08:00 SRC=10.255.0.1 DST=10.10.10.10 LEN=29 TOS=0x00 PREC=0x00 TTL=63 ID=2806 DF PROTO=UDP SPT=36556 DPT=11111 LEN=9 MARK=0x1\n"
	EGRESS_DENIED_KERNEL_LOG := "Jun 30 16:07:06 localhost kernel: [265213.303412] DENY_container-handle-1-long IN=s-010255095010 OUT=eth0 MAC=aa:aa:0a:ff:5f:0a:ee:ee:0a:ff:5f:0a:08:00 SRC=10.255.0.1 DST=10.10.10.10 LEN=30 TOS=0x00 PREC=0x00 TTL=63 ID=2535 DF PROTO=UDP SPT=45564 DPT=25555 LEN=10 MARK=0x1\n"

	BeforeEach(func() {
		var err error
		kernelLogFile, err = ioutil.TempFile("", "")
		Expect(err).ToNot(HaveOccurred())

		containerMetadataFile, err = ioutil.TempFile("", "")
		Expect(err).ToNot(HaveOccurred())

		outputDir, err := ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())

		outputFile = filepath.Join(outputDir, "iptables.log")
		conf = config.LogTransformer{
			KernelLogFile:         kernelLogFile.Name(),
			ContainerMetadataFile: containerMetadataFile.Name(),
			OutputLogFile:         outputFile,
		}
		configFilePath = WriteConfigFile(conf)

		logTransformerCmd := exec.Command(binaryPath, "-config-file", configFilePath)
		session, err = gexec.Start(logTransformerCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		store = &datastore.Store{
			Serializer: &serial.Serial{},
			Locker:     filelock.NewLocker(containerMetadataFile.Name()),
		}
		AddToContainerMetadata(store, "container-handle-1-longer-than-29-chars", "10.255.0.1", map[string]interface{}{
			"org_id":          "organization_id_1",
			"space_id":        "space_id_1",
			"app_id":          "app_id_1",
			"policy_group_id": "policy_group_id_1",
		})
		AddToContainerMetadata(store, "container-handle-2-longer-than-29-chars", "10.255.0.2", map[string]interface{}{
			"org_id":          "organization_id_1",
			"space_id":        "space_id_2",
			"app_id":          "app_id_2",
			"policy_group_id": "policy_group_id_2",
		})
	})

	AfterEach(func() {
		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
	})

	It("should log when starting", func() {
		Eventually(session.Out).Should(gbytes.Say("cfnetworking.log-transformer.*starting"))
	})

	It("should run as a daemon", func() {
		Consistently(session, DEFAULT_TIMEOUT).ShouldNot(gexec.Exit())
	})

	It("should not truncate output log file on restart", func() {
		go AddToKernelLog(EGRESS_ALLOWD_KERNEL_LOG, kernelLogFile)
		Eventually(outputFile).Should(BeAnExistingFile())

		Eventually(func() string {
			bytes, err := ioutil.ReadFile(outputFile)
			Expect(err).NotTo(HaveOccurred())
			return string(bytes)
		}, "5s").ShouldNot(BeEmpty())

		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())

		var err error
		logTransformerCmd := exec.Command(binaryPath, "-config-file", configFilePath)
		session, err = gexec.Start(logTransformerCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(outputFile).Should(BeAnExistingFile())

		go AddToKernelLog(EGRESS_DENIED_KERNEL_LOG, kernelLogFile)

		Eventually(ReadLines, "5s").Should(ContainElement(MatchJSON(EGRESS_DENIED_JSON)))
		Eventually(ReadLines, "5s").Should(ContainElement(MatchJSON(EGRESS_ALLOWED_JSON)))
	})

	It("logs data about packets", func() {
		By("logging successful egress packets")
		go AddToKernelLog(EGRESS_ALLOWD_KERNEL_LOG, kernelLogFile)
		Eventually(outputFile).Should(BeAnExistingFile())
		Eventually(ReadLines, "5s").Should(ContainElement(MatchJSON(EGRESS_ALLOWED_JSON)))

		By("logging denied egress packets")
		go AddToKernelLog(EGRESS_DENIED_KERNEL_LOG, kernelLogFile)
		Eventually(outputFile).Should(BeAnExistingFile())
		Eventually(ReadLines, "5s").Should(ContainElement(MatchJSON(EGRESS_DENIED_JSON)))
	})

	Context("when source file is rotated", func() {
		It("logs data about packets", func() {
			By("logging successful egress packets")
			go AddToKernelLog(EGRESS_ALLOWD_KERNEL_LOG, kernelLogFile)
			Eventually(outputFile).Should(BeAnExistingFile())
			Eventually(ReadLines, "5s").Should(ContainElement(MatchJSON(EGRESS_ALLOWED_JSON)))
			kernelLogFilename := kernelLogFile.Name()

			By("rotate source file")
			var err error
			Expect(os.Rename(kernelLogFile.Name(), filepath.Join(os.TempDir(), "kernel.log.backup"))).To(Succeed())
			kernelLogFile, err = os.Create(kernelLogFilename)
			Expect(err).ToNot(HaveOccurred())

			By("logging denied egress packets")
			go AddToKernelLog(EGRESS_DENIED_KERNEL_LOG, kernelLogFile)
			Eventually(outputFile).Should(BeAnExistingFile())
			Eventually(ReadLines, "5s").Should(ContainElement(MatchJSON(EGRESS_DENIED_JSON)))
		})
	})

	Context("when source file is removed", func() {
		It("logs data about packets", func() {
			By("logging successful egress packets")
			go AddToKernelLog(EGRESS_ALLOWD_KERNEL_LOG, kernelLogFile)
			Eventually(outputFile).Should(BeAnExistingFile())
			Eventually(ReadLines, "5s").Should(ContainElement(MatchJSON(EGRESS_ALLOWED_JSON)))
			kernelLogFilename := kernelLogFile.Name()

			By("remove source file")
			var err error
			Expect(os.Remove(kernelLogFile.Name())).To(Succeed())
			kernelLogFile, err = os.Create(kernelLogFilename)
			Expect(err).ToNot(HaveOccurred())

			By("logging denied egress packets")
			go AddToKernelLog(EGRESS_DENIED_KERNEL_LOG, kernelLogFile)
			Eventually(outputFile).Should(BeAnExistingFile())
			Eventually(ReadLines, "5s").Should(ContainElement(MatchJSON(EGRESS_DENIED_JSON)))
		})
	})

	Context("when destination file is rotated", func() {
		It("logs data about packets", func() {
			By("logging successful egress packets")
			go AddToKernelLog(EGRESS_ALLOWD_KERNEL_LOG, kernelLogFile)
			Eventually(outputFile).Should(BeAnExistingFile())
			Eventually(ReadLines, "5s").Should(ContainElement(MatchJSON(EGRESS_ALLOWED_JSON)))

			By("rotate destination file")
			Expect(os.Rename(outputFile, filepath.Join(os.TempDir(), "destination.log.backup"))).To(Succeed())
			_, err := os.Create(outputFile)
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the rotatable sink to pickup the new file")
			time.Sleep(2 * time.Second)

			By("logging denied egress packets")
			go AddToKernelLog(EGRESS_DENIED_KERNEL_LOG, kernelLogFile)
			Eventually(outputFile).Should(BeAnExistingFile())
			Eventually(ReadLines, "5s").Should(ContainElement(MatchJSON(EGRESS_DENIED_JSON)))
		})
	})

	Context("when destination file is removed", func() {
		It("logs data about packets", func() {
			By("logging successful egress packets")
			go AddToKernelLog(EGRESS_ALLOWD_KERNEL_LOG, kernelLogFile)
			Eventually(outputFile).Should(BeAnExistingFile())
			Eventually(ReadLines, "5s").Should(ContainElement(MatchJSON(EGRESS_ALLOWED_JSON)))

			By("remove destination file")
			Expect(os.Remove(outputFile)).To(Succeed())

			By("waiting for the rotatable sink to pickup the new file")
			time.Sleep(2 * time.Second)

			By("logging denied egress packets")
			go AddToKernelLog(EGRESS_DENIED_KERNEL_LOG, kernelLogFile)
			Eventually(outputFile, "5s").Should(BeAnExistingFile())
			Eventually(ReadLines, "5s").Should(ContainElement(MatchJSON(EGRESS_DENIED_JSON)))
		})
	})
})

func AddToContainerMetadata(store *datastore.Store, containerID, containerIP string, metadata map[string]interface{}) {
	err := store.Add(containerID, containerIP, metadata)
	Expect(err).NotTo(HaveOccurred())
}
func AddToKernelLog(line string, w io.Writer) {
	defer GinkgoRecover()

	time.Sleep(200 * time.Millisecond)
	_, err := w.Write([]byte(line))
	Expect(err).NotTo(HaveOccurred())
}

func ReadLines() []string {
	output := strings.Split(ReadOutput(), "\n")
	output = output[:len(output)-1]

	var outputs []string
	for _, o := range output {
		var outputMap map[string]interface{}
		err := json.Unmarshal([]byte(o), &outputMap)
		Expect(err).NotTo(HaveOccurred())

		outputMap["timestamp"] = "some-timestamp"
		outputJson, err := json.Marshal(outputMap)
		Expect(err).NotTo(HaveOccurred())

		outputs = append(outputs, string(outputJson))
	}

	return outputs
}

func ReadOutput() string {
	bytes, err := ioutil.ReadFile(outputFile)
	Expect(err).NotTo(HaveOccurred())
	if string(bytes) == "" {
		return "{}"
	}
	return string(bytes)
}
