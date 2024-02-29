package integration_test

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"
	"code.cloudfoundry.org/policy-server/config"
	"code.cloudfoundry.org/policy-server/integration/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("External API Tags", func() {
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
		dbConf.DatabaseName = fmt.Sprintf("external_api_tags_test_node_%d", ports.PickAPort())

		template, _, _ := helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "fixtures")
		policyServerConfs = configurePolicyServers(template, 2)
		sessions = startPolicyServers(policyServerConfs)
		conf = policyServerConfs[0]

		body := strings.NewReader(`{ "policies": [
			{"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "ports": { "start": 8090, "end": 8090 } } },
			{"source": { "id": "some-app-guid" }, "destination": { "id": "another-app-guid", "protocol": "udp", "ports": { "start": 6666, "end": 6666 } } },
			{"source": { "id": "another-app-guid" }, "destination": { "id": "some-app-guid", "protocol": "tcp", "ports": { "start": 3333, "end": 3333 } } }
			] }`)
		resp := helpers.MakeAndDoRequest(
			"POST",
			fmt.Sprintf("http://%s:%d/networking/v1/external/policies", conf.ListenHost, conf.ListenPort),
			nil,
			body,
		)

		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		responseString, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(responseString).To(MatchJSON("{}"))
	})

	AfterEach(func() {
		stopPolicyServers(sessions, policyServerConfs)

		Expect(fakeMetron.Close()).To(Succeed())
	})

	Describe("listing tags", func() {
		listingTagsSucceeds := func(version string) {
			By("listing the current tags")
			resp := helpers.MakeAndDoRequest(
				"GET",
				fmt.Sprintf("http://%s:%d/networking/%s/external/tags", conf.ListenHost, conf.ListenPort, version),
				nil,
				nil,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseString).To(MatchJSON(`{ "tags": [
				{ "id": "some-app-guid", "tag": "01", "type": "app" },
				{ "id": "some-other-app-guid", "tag": "02", "type": "app" },
				{ "id": "another-app-guid", "tag": "03", "type": "app" }
			] }`))

			By("reusing tags that are no longer in use")
			body := strings.NewReader(`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "ports": { "start": 8090, "end": 8090 } } } ] }`)
			helpers.MakeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/v1/external/policies/delete", conf.ListenHost, conf.ListenPort),
				nil,
				body,
			)

			body = strings.NewReader(`{ "policies": [ {"source": { "id": "some-app-guid" }, "destination": { "id": "yet-another-app-guid", "protocol": "udp", "ports": { "start": 4567, "end": 4567 } } } ] }`)
			helpers.MakeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/networking/v1/external/policies", conf.ListenHost, conf.ListenPort),
				nil,
				body,
			)

			resp = helpers.MakeAndDoRequest(
				"GET",
				fmt.Sprintf("http://%s:%d/networking/%s/external/tags", conf.ListenHost, conf.ListenPort, version),
				nil,
				nil,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err = io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseString).To(MatchJSON(`{ "tags": [
				{ "id": "some-app-guid", "tag": "01", "type": "app" },
				{ "id": "yet-another-app-guid", "tag": "02", "type": "app" },
				{ "id": "another-app-guid", "tag": "03", "type": "app" }
			] }`))

			Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
				HaveName("TagsIndexRequestTime"),
			))
			Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
				HaveName("StoreTagsSuccessTime"),
			))
		}

		DescribeTable("listing tags succeeds", listingTagsSucceeds,
			Entry("v1", "v1"),
			Entry("v0", "v0"),
		)
	})
})
