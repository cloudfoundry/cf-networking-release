package integration_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"policy-server/config"
	"policy-server/integration/helpers"
	"strings"

	"code.cloudfoundry.org/go-db-helpers/metrics"
	"code.cloudfoundry.org/go-db-helpers/testsupport"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Internal API", func() {
	var (
		sessions     []*gexec.Session
		conf         config.Config
		address      string
		testDatabase *testsupport.TestDatabase
		tlsConfig    *tls.Config

		fakeMetron metrics.FakeMetron
	)

	BeforeEach(func() {
		fakeMetron = metrics.NewFakeMetron()
		dbName := fmt.Sprintf("test_netman_database_%x", rand.Int())
		dbConnectionInfo := testsupport.GetDBConnectionInfo()
		testDatabase = dbConnectionInfo.CreateDatabase(dbName)
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

		template := helpers.DefaultTestConfig(testDatabase.DBConfig(), fakeMetron.Address(), "fixtures")
		template.TagLength = 2
		policyServerConfs := configurePolicyServers(template, 1)
		sessions = startPolicyServers(policyServerConfs)
		conf = policyServerConfs[0]

		address = fmt.Sprintf("%s:%d", conf.ListenHost, conf.ListenPort)
	})

	AfterEach(func() {
		stopPolicyServers(sessions)

		if testDatabase != nil {
			testDatabase.Destroy()
		}

		Expect(fakeMetron.Close()).To(Succeed())
	})

	It("Lists policies and associated tags", func() {
		body := strings.NewReader(`{ "policies": [
				 {"source": { "id": "app1" }, "destination": { "id": "app2", "protocol": "tcp", "port": 8080 } },
				 {"source": { "id": "app3" }, "destination": { "id": "app1", "protocol": "tcp", "port": 9999 } },
				 {"source": { "id": "app3" }, "destination": { "id": "app4", "protocol": "tcp", "port": 3333 } }
				 ]}
				`)

		_ = helpers.MakeAndDoRequest(
			"POST",
			fmt.Sprintf("http://%s:%d/networking/v0/external/policies", conf.ListenHost, conf.ListenPort),
			body,
		)

		resp := helpers.MakeAndDoHTTPSRequest(
			"GET",
			fmt.Sprintf("https://%s:%d/networking/v0/internal/policies?id=app1,app2", conf.ListenHost, conf.InternalListenPort),
			nil,
			tlsConfig,
		)
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		responseString, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(responseString).To(MatchJSON(`{ "policies": [
				{"source": { "id": "app1", "tag": "0001" }, "destination": { "id": "app2", "tag": "0002", "protocol": "tcp", "port": 8080 } },
				{"source": { "id": "app3", "tag": "0003" }, "destination": { "id": "app1", "tag": "0001", "protocol": "tcp", "port": 9999 } }
			]}
		`))
	})

	It("emits metrics about durations", func() {
		resp := helpers.MakeAndDoHTTPSRequest(
			"GET",
			fmt.Sprintf("https://%s:%d/networking/v0/internal/policies?id=app1,app2", conf.ListenHost, conf.InternalListenPort),
			nil,
			tlsConfig,
		)
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
			HaveName("InternalPoliciesRequestTime"),
		))
		Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
			HaveName("StoreAllTime"),
		))
	})
})
