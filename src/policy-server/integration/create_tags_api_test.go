package integration_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"policy-server/config"
	"policy-server/integration/helpers"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"

	"net/http"
	"strings"

	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Create Tags API", func() {
	var (
		sessions                  []*gexec.Session
		conf                      config.Config
		internalConf              config.InternalConfig
		dbConf                    db.Config
		tlsConfig                 *tls.Config
		policyServerConfs         []config.Config
		policyServerInternalConfs []config.InternalConfig
		fakeMetron                metrics.FakeMetron
	)

	BeforeEach(func() {
		fakeMetron = metrics.NewFakeMetron()
		dbConf = testsupport.GetDBConfig()
		dbConf.Timeout = 5
		dbConf.DatabaseName = fmt.Sprintf("internal_api_test_node_%d", ports.PickAPort())

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
		policyServerConfs = configurePolicyServers(template, 1)
		policyServerInternalConfs = configureInternalPolicyServers(internalTemplate, 1)
		sessions = startPolicyAndInternalServers(policyServerConfs, policyServerInternalConfs)
		conf = policyServerConfs[0]
		internalConf = policyServerInternalConfs[0]
	})

	AfterEach(func() {
		stopPolicyServerExternalAndInternal(sessions, policyServerConfs, policyServerInternalConfs)
		Expect(fakeMetron.Close()).To(Succeed())
	})

	Context("when the id has not been used before", func() {
		var resp *http.Response
		BeforeEach(func() {
			body := strings.NewReader(`{"type": "router-type", "id": "router-guid" }`)

			resp = helpers.MakeAndDoHTTPSRequest(
				"PUT",
				fmt.Sprintf("https://%s:%d/networking/v1/internal/tags", internalConf.ListenHost, internalConf.InternalListenPort),
				body,
				tlsConfig,
			)
		})

		It("creates a new tag", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseBody, err := ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(responseBody)).To(MatchJSON(`{"type":"router-type","id":"router-guid","tag":"0001"}`))
		})

		Context("when creating a tag with the same parameters", func() {
			BeforeEach(func() {
				body := strings.NewReader(`{"type": "router-type", "id": "router-guid" }`)

				resp = helpers.MakeAndDoHTTPSRequest(
					"PUT",
					fmt.Sprintf("https://%s:%d/networking/v1/internal/tags", internalConf.ListenHost, internalConf.InternalListenPort),
					body,
					tlsConfig,
				)
			})

			It("returns the same tag", func() {
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseBody, err := ioutil.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(responseBody)).To(MatchJSON(`{"type":"router-type","id":"router-guid","tag":"0001"}`))
			})
		})

		Context("when creating a tag with the same guid and a different type", func() {
			It("fails", func() {
				body := strings.NewReader(`{"type": "meow", "id": "router-guid" }`)

				resp = helpers.MakeAndDoHTTPSRequest(
					"PUT",
					fmt.Sprintf("https://%s:%d/networking/v1/internal/tags", internalConf.ListenHost, internalConf.InternalListenPort),
					body,
					tlsConfig,
				)

				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
			})
		})
	})
})
