package performance_test

import (
	helpersConfig "github.com/cloudfoundry-incubator/cf-test-helpers/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"io/ioutil"
	"testing"
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
