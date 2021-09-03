package integration_test

import (
	"crypto/tls"
	"fmt"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"
	"code.cloudfoundry.org/policy-server/config"
	"code.cloudfoundry.org/policy-server/integration/helpers"
	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Strict-Transport-Security Header", func() {
	var (
		sessions          []*gexec.Session
		conf              config.Config
		policyServerConfs []config.Config
		dbConf            db.Config

		fakeMetron metrics.FakeMetron
	)

	BeforeEach(func() {
		fakeMetron = metrics.NewFakeMetron()

		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("cors_test_node_%d", ports.PickAPort())

		template, _ := helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "fixtures")
		policyServerConfs = configurePolicyServers(template, 1)
	})

	JustBeforeEach(func() {
		sessions = startPolicyServers(policyServerConfs)
		conf = policyServerConfs[0]
	})

	AfterEach(func() {
		stopPolicyServers(sessions, policyServerConfs)

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Context("when TLS is disabled", func() {
		It("should not add Strict-Transport-Security header in the response", func() {
			resp := helpers.MakeAndDoRequest(
				"GET",
				fmt.Sprintf("http://%s:%d/networking/v1/external/policies", conf.ListenHost, conf.ListenPort),
				nil,
				nil,
			)

			Expect(resp.Header.Get("Strict-Transport-Security")).To(Equal(""))
		})
	})

	Context("when TLS is enabled", func() {
		var (
			tlsConfig *tls.Config
		)

		BeforeEach(func() {
			policyServerConfs[0].EnableTLS = true
			tlsConfig = helpers.DefaultTLSConfig()
		})

		It("returns Strict-Transport-Security headers when an endpoint is hit", func() {
			resp := helpers.MakeAndDoHTTPSRequest(
				"GET",
				fmt.Sprintf("https://%s:%d/networking/v1/external/policies", conf.ListenHost, conf.ListenPort),
				nil,
				tlsConfig,
			)

			Expect(resp.Header.Get("Strict-Transport-Security")).To(Equal("max-age=31536000"))
		})
	})
})
