package integration_test

import (
	"fmt"
	"policy-server/config"
	"policy-server/integration/helpers"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"
	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cross Origin Resource Sharing", func() {
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
		template.AllowedCORSDomains = []string{
			"foo.bar",
			"bar.foo",
		}
		policyServerConfs = configurePolicyServers(template, 2)
		sessions = startPolicyServers(policyServerConfs)
		conf = policyServerConfs[0]
	})

	AfterEach(func() {
		stopPolicyServers(sessions, policyServerConfs, nil)

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Context("when a user makes a cors preflight request", func() {
		It("returns cors headers", func() {
			resp := helpers.MakeAndDoRequest(
				"OPTIONS",
				fmt.Sprintf("http://%s:%d/", conf.ListenHost, conf.ListenPort),
				nil,
				nil,
			)

			Expect(resp.Header["Access-Control-Allow-Origin"]).To(ContainElement("foo.bar,bar.foo"))
			Expect(resp.Header["Access-Control-Allow-Methods"]).To(ContainElement("GET,OPTIONS"))
		})
	})

	Context("when a user makes a GET request", func() {
		It("returns cors allow origin header", func() {
			resp := helpers.MakeAndDoRequest(
				"GET",
				fmt.Sprintf("http://%s:%d/", conf.ListenHost, conf.ListenPort),
				nil,
				nil,
			)

			Expect(resp.Header["Access-Control-Allow-Origin"]).To(ContainElement("foo.bar,bar.foo"))
		})
	})
})
