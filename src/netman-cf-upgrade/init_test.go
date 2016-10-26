package netman_cf_upgrade_test

import (
	"cf-pusher/cf_cli_adapter"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"testing"
)

const Timeout_Short = 10 * time.Second
const BOSH_DEPLOY_TIMEOUT = 75 * time.Minute

var (
	config     helpers.Config
	boshConfig *BoshConfig
	cli        *cf_cli_adapter.Adapter
)

type BoshConfig struct {
	DirectorURL    string `json:"bosh_director_url"`
	AdminUser      string `json:"bosh_admin_user"`
	AdminPassword  string `json:"bosh_admin_password"`
	DeploymentName string `json:"bosh_deployment_name"`
}

func TestNetmanCfUpgrade(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NetmanCfUpgrade Suite")
}

var _ = BeforeSuite(func() {
	bytes, err := ioutil.ReadFile(os.Getenv("CONFIG"))
	Expect(err).NotTo(HaveOccurred())
	boshConfig = &BoshConfig{}
	err = json.Unmarshal(bytes, boshConfig)
	Expect(err).NotTo(HaveOccurred(), "Could not unmarshal config file. Make sure it is valid JSON.")
	config = helpers.LoadConfig()
	cli = &cf_cli_adapter.Adapter{CfCliPath: "cf"}
})

func boshCmd(manifest, action, completeMsg string) {
	args := []string{"-n"}
	if manifest != "" {
		args = append(args, "-d", manifest)
	}
	args = append(args, strings.Split(action, " ")...)
	cmd := bosh(args...)
	sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, BOSH_DEPLOY_TIMEOUT).Should(gexec.Exit(0))
	Expect(sess).To(gbytes.Say(completeMsg))
}

func bosh(args ...string) *exec.Cmd {
	boshArgs := append([]string{"-t", boshConfig.DirectorURL, "-u", boshConfig.AdminUser, "-p", boshConfig.AdminPassword}, args...)
	return exec.Command("bosh", boshArgs...)
}
