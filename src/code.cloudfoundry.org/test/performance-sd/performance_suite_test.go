package performance_test

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	helpersConfig "github.com/cloudfoundry/cf-test-helpers/v2/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
	})
	RunSpecs(t, "Performance Suite")
}
