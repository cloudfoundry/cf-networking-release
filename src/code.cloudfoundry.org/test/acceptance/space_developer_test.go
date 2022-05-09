package acceptance_test

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/policy-server/api/api_v0"
	"code.cloudfoundry.org/policy_client"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/warrant"
)

var _ = Describe("space developer policy configuration", func() {
	var (
		appA       string
		appB       string
		orgName    string
		spaceNameA string
		spaceNameB string
		prefix     string

		policyClient *policy_client.ExternalClient

		warrantClient warrant.Warrant
	)

	BeforeEach(func() {
		policyClient = policy_client.NewExternal(lagertest.NewTestLogger("test"),
			&http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			},
			fmt.Sprintf("https://%s", config.ApiEndpoint),
		)

		warrantClient = warrant.New(warrant.Config{
			Host:          getUAABaseURL(),
			SkipVerifySSL: true,
		})

		prefix = testConfig.Prefix
		appA = fmt.Sprintf("appA-%d", rand.Int31())
		appB = fmt.Sprintf("appB-%d", rand.Int31())

		AuthAsAdmin()

		orgName = prefix + "space-developer-org"
		Expect(cf.Cf("create-org", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))

		spaceNameA = prefix + "space-A"
		Expect(cf.Cf("create-space", spaceNameA, "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName, "-s", spaceNameA).Wait(Timeout_Push)).To(gexec.Exit(0))

		pushProxy(appA)

		spaceNameB = prefix + "space-B"
		Expect(cf.Cf("create-space", spaceNameB, "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName, "-s", spaceNameB).Wait(Timeout_Push)).To(gexec.Exit(0))

		pushProxy(appB)

		uaaAdminClientToken, err := warrantClient.Clients.GetToken("admin", testConfig.AdminSecret)
		Expect(err).NotTo(HaveOccurred())

		user := ensureUserExists(warrantClient, "space-developer", "password", uaaAdminClientToken)
		group := ensureGroupExists(warrantClient, "network.write", uaaAdminClientToken)

		_, err = warrantClient.Groups.AddMember(group.ID, user.ID, uaaAdminClientToken)
		Expect(err).To(Or(BeNil(), BeAssignableToTypeOf(warrant.DuplicateResourceError{})))

		Expect(cf.Cf("set-space-role", "space-developer", orgName, spaceNameA, "SpaceDeveloper").Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("set-space-role", "space-developer", orgName, spaceNameB, "SpaceDeveloper").Wait(Timeout_Push)).To(gexec.Exit(0))

		err = warrantClient.Clients.Create(warrant.Client{
			ID:                   "space-client",
			Name:                 "space-client",
			Authorities:          []string{"network.write", "cloud_controller.read"},
			AuthorizedGrantTypes: []string{"client_credentials"},
			AccessTokenValidity:  600 * time.Second,
		}, "password", uaaAdminClientToken)
		Expect(err).NotTo(HaveOccurred())

		orgGuid, err := cfCLI.OrgGuid(orgName)
		Expect(err).NotTo(HaveOccurred())

		spaceAGuid, err := cfCLI.SpaceGuid(spaceNameA)
		Expect(err).NotTo(HaveOccurred())

		spaceBGuid, err := cfCLI.SpaceGuid(spaceNameB)
		Expect(err).NotTo(HaveOccurred())

		Expect(cf.Cf("curl", "-X", "PUT", fmt.Sprintf("/v2/organizations/%s/users/space-client", orgGuid)).Wait()).To(gexec.Exit(0))

		cf.Cf("curl", "-X", "PUT", fmt.Sprintf("/v2/spaces/%s/developers/space-client", spaceAGuid))
		cf.Cf("curl", "-X", "PUT", fmt.Sprintf("/v2/spaces/%s/developers/space-client", spaceBGuid))
	})

	AfterEach(func() {
		By("logging in as admin and deleting the org", func() {
			Expect(cf.Cf("logout").Wait(Timeout_Push)).To(gexec.Exit(0))
			Expect(cf.Cf("auth", config.AdminUser, config.AdminPassword).Wait(Timeout_Push)).To(gexec.Exit(0))

			uaaAdminClientToken, err := warrantClient.Clients.GetToken("admin", testConfig.AdminSecret)
			Expect(err).NotTo(HaveOccurred())
			warrantClient.Clients.Delete("space-client", uaaAdminClientToken)

			Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
		})
	})

	Describe("space developer with network.write scope", func() {
		DescribeTable("can create, list, and delete network policies in spaces they have access to", func(authArgs []string) {
			var spaceDevUserToken string
			By("logging in and getting the space developer user token", func() {
				authArgs = append([]string{"auth"}, authArgs...)
				Expect(cf.Cf(authArgs...).Wait(Timeout_Push)).To(gexec.Exit(0))
				session := cf.Cf("oauth-token")
				Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))
				spaceDevUserToken = strings.TrimSpace(string(session.Out.Contents()))
			})

			var appAGUID, appBGUID string
			By("getting the app guids", func() {
				Expect(cf.Cf("target", "-o", orgName, "-s", spaceNameA).Wait(Timeout_Push)).To(gexec.Exit(0))
				session := cf.Cf("app", appA, "--guid")
				Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))
				appAGUID = strings.TrimSpace(string(session.Out.Contents()))

				Expect(cf.Cf("target", "-o", orgName, "-s", spaceNameB).Wait(Timeout_Push)).To(gexec.Exit(0))
				session = cf.Cf("app", appB, "--guid")
				Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))
				appBGUID = strings.TrimSpace(string(session.Out.Contents()))
			})

			By("creating a policy", func() {
				err := policyClient.AddPoliciesV0(spaceDevUserToken, []api_v0.Policy{
					{
						Source: api_v0.Source{
							ID: appAGUID,
						},
						Destination: api_v0.Destination{
							ID:       appBGUID,
							Port:     1234,
							Protocol: "tcp",
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())
			})

			By("listing policies", func() {
				expectedPolicies := []api_v0.Policy{
					{
						Source: api_v0.Source{
							ID: appAGUID,
						},
						Destination: api_v0.Destination{
							ID:       appBGUID,
							Port:     1234,
							Protocol: "tcp",
						},
					},
				}
				policies, err := policyClient.GetPoliciesV0(spaceDevUserToken)
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(Equal(expectedPolicies))
			})

			By("deleting the policy", func() {
				err := policyClient.DeletePoliciesV0(spaceDevUserToken, []api_v0.Policy{
					{
						Source: api_v0.Source{
							ID: appAGUID,
						},
						Destination: api_v0.Destination{
							ID:       appBGUID,
							Port:     1234,
							Protocol: "tcp",
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())
			})
		},
			Entry("as a user", []string{"space-developer", "password"}),
			Entry("as a service account", []string{"space-client", "password", "--client-credentials"}),
		)
	})
})

func ensureGroupExists(client warrant.Warrant, name, token string) warrant.Group {
	_, err := client.Groups.Create(name, token)
	Expect(err).To(Or(BeNil(), BeAssignableToTypeOf(warrant.DuplicateResourceError{})))

	groups, err := client.Groups.List(warrant.Query{Filter: fmt.Sprintf(`displayName eq %q`, name)}, token)
	Expect(err).NotTo(HaveOccurred())

	return groups[0]
}

func ensureUserExists(warrantClient warrant.Warrant, username, password, token string) warrant.User {
	createUserSession := cf.Cf("create-user", username, password)
	Expect(createUserSession.Wait(Timeout_Push)).To(gexec.Exit(0))

	users, err := warrantClient.Users.List(warrant.Query{Filter: fmt.Sprintf(`userName eq %q`, username)}, token)
	Expect(err).NotTo(HaveOccurred())

	return users[0]
}
