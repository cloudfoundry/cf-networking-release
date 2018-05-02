package deploy_test

import (
	helpersConfig "github.com/cloudfoundry-incubator/cf-test-helpers/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"testing"
	"time"

	"os"

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
			tempVarsStore, err := ioutil.TempFile("", "")
			Expect(err).ToNot(HaveOccurred())

			_, err = tempVarsStore.Write([]byte(fmt.Sprintf(
				`nats_password: %s
nats_ip: %s`, config.NatsPassword, config.NatsURL)))
			Expect(err).ToNot(HaveOccurred())

			cmd := exec.Command("bosh", "deploy", "-n", "-d", "performance", "../test_assets/manifest.yml", "--vars-store", tempVarsStore.Name())
			session, err := gexec.Start(cmd, os.Stdout, os.Stderr)
			Expect(err).ToNot(HaveOccurred())

			Eventually(session, 20*time.Minute).Should(gexec.Exit(0))
		})
	})
	RunSpecs(t, "Deploy")
}
