package integration_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"policy-server/config"
	"policy-server/integration/helpers"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Internal API", func() {
	var (
		sessions     []*gexec.Session
		conf         config.Config
		internalConf config.InternalConfig
		address      string
		dbConf       db.Config
		tlsConfig    *tls.Config

		fakeMetron testsupport.FakeMetron
	)

	BeforeEach(func() {
		fakeMetron = testsupport.NewFakeMetron()
		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("test_node_%d", GinkgoParallelNode())
		testsupport.CreateDatabase(dbConf)

		cert, err := tls.LoadX509KeyPair("fixtures/client.crt", "fixtures/client.key")
		Expect(err).NotTo(HaveOccurred())

		clientCACert, err := ioutil.ReadFile("fixtures/netman-ca.crt")
		Expect(err).NotTo(HaveOccurred())

		clientCertPool := x509.NewCertPool()
		clientCertPool.AppendCertsFromPEM(clientCACert)

		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      clientCertPool,
		}
		tlsConfig.BuildNameToCertificate()

		template, internalTemplate := helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "fixtures")
		template.TagLength = 2
		internalTemplate.TagLength = 2
		policyServerConfs := configurePolicyServers(template, 1)
		policyServerInternalConfs := configureInternalPolicyServers(internalTemplate, 1)
		sessions = startPolicyAndInternalServers(policyServerConfs, policyServerInternalConfs)
		conf = policyServerConfs[0]
		internalConf = policyServerInternalConfs[0]

		address = fmt.Sprintf("%s:%d", conf.ListenHost, conf.ListenPort)

		body := strings.NewReader(`{ "policies": [
				 {"source": { "id": "app1" }, "destination": { "id": "app2", "protocol": "tcp", "ports": { "start": 8080, "end": 8080 } } },
				 {"source": { "id": "app3" }, "destination": { "id": "app1", "protocol": "tcp", "ports": { "start": 9999, "end": 9999 } } },
				 {"source": { "id": "app3" }, "destination": { "id": "app2", "protocol": "tcp", "ports": { "start": 3333, "end": 4444 } } },
				 {"source": { "id": "app3" }, "destination": { "id": "app4", "protocol": "tcp", "ports": { "start": 3333, "end": 3333 } } }
				 ]}
				`)

		_ = helpers.MakeAndDoRequest(
			"POST",
			fmt.Sprintf("http://%s:%d/networking/v1/external/policies", conf.ListenHost, conf.ListenPort),
			nil,
			body,
		)

	})

	AfterEach(func() {
		stopPolicyServers(sessions)

		testsupport.RemoveDatabase(dbConf)

		Expect(fakeMetron.Close()).To(Succeed())
	})

	listPoliciesAndTagsSucceeds := func(version, expectedResponse string) {
		resp := helpers.MakeAndDoHTTPSRequest(
			"GET",
			fmt.Sprintf("https://%s:%d/networking/%s/internal/policies?id=app1,app2", internalConf.ListenHost, internalConf.InternalListenPort, version),
			nil,
			tlsConfig,
		)
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		responseString, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(responseString).To(MatchJSON(expectedResponse))
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
			HaveName("InternalPoliciesRequestTime"),
		))
		Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
			HaveName("StoreAllSuccessTime"),
		))
	}

	v1Response := `{"total_policies": 3,
			"policies": [
			{"source": { "id": "app1", "tag": "0001" }, "destination": { "id": "app2", "tag": "0002", "protocol": "tcp", "ports": {"start": 8080, "end": 8080 } } },
			{"source": { "id": "app3", "tag": "0003" }, "destination": { "id": "app1", "tag": "0001", "protocol": "tcp", "ports": {"start": 9999, "end": 9999 } } },
			{"source": { "id": "app3", "tag": "0003" }, "destination": { "id": "app2", "tag": "0002", "protocol": "tcp", "ports": { "start": 3333, "end": 4444 } } }
			]}`

	v0Response := `{"total_policies": 2,
			"policies": [
			{"source": { "id": "app1", "tag": "0001" }, "destination": { "id": "app2", "tag": "0002", "protocol": "tcp", "port": 8080, "ports": {"start": 8080, "end": 8080 } } },
			{"source": { "id": "app3", "tag": "0003" }, "destination": { "id": "app1", "tag": "0001", "protocol": "tcp", "port": 9999, "ports": {"start": 9999, "end": 9999 } } }
			]}`

	DescribeTable("listing policies and tags succeeds", listPoliciesAndTagsSucceeds,
		Entry("v1", "v1", v1Response),
		Entry("v0", "v0", v0Response),
	)

})
