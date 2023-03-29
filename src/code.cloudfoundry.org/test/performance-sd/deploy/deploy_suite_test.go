package deploy_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	helpersConfig "github.com/cloudfoundry/cf-test-helpers/v2/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const NATS_MSG_SIZE = 1024

var (
	config Config
)

type Config struct {
	NatsURL            string `json:"nats_url"`
	NatsUsername       string `json:"nats_username"`
	NatsPassword       string `json:"nats_password"`
	NatsMonitoringPort int    `json:"nats_monitoring_port"`
	NatsPort           int    `json:"nats_port"`
	NumMessages        int    `json:"num_messages"`
	NumPublisher       int    `json:"num_publishers"`
}

func TestPerformance(t *testing.T) {
	RegisterFailHandler(Fail)
	BeforeSuite(func() {
		// Read and set config
		configPath := helpersConfig.ConfigPath()
		configBytes, err := ioutil.ReadFile(configPath)
		Expect(err).NotTo(HaveOccurred())

		err = json.Unmarshal(configBytes, &config)
		Expect(err).NotTo(HaveOccurred())

		By("deploying bosh-dns, bosh-dns-adapter, and service-discovery-controller, nats", func() {
			cmd := exec.Command("bosh", "deploy", "-n", "-d", "performance", "../test_assets/manifest.yml",
				"-v", fmt.Sprintf("nats_password=%s", config.NatsPassword),
				"-v", fmt.Sprintf("nats_ip=%s", config.NatsURL))
			session, err := gexec.Start(cmd, os.Stdout, os.Stderr)
			Expect(err).ToNot(HaveOccurred())

			Eventually(session, 20*time.Minute).Should(gexec.Exit(0))
		})
	})
	RunSpecs(t, "Deploy")
}
