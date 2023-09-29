package acceptance_test

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"code.cloudfoundry.org/lager/v3/lagertest"
	"code.cloudfoundry.org/policy_client"
	uaa "github.com/cloudfoundry-community/go-uaa"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("space developer policy configuration", func() {
	var (
		appA       string
		appB       string
		spaceNameA string
		spaceNameB string

		policyClient *policy_client.ExternalClient
	)

	Describe("space developer with network.write scope", func() {
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

			testConfig.Prefix = fmt.Sprintf("%s%d-", testConfig.Prefix, rand.Int31())
			appA = fmt.Sprintf("appA-%d", rand.Int31())
			appB = fmt.Sprintf("appB-%d", rand.Int31())

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Timeout_Push, func() {
				Expect(cf.Cf("target", "-o", TestSetup.TestSpace.OrganizationName()).Wait(Timeout_Push)).To(gexec.Exit(0))

				spaceNameA = fmt.Sprintf("%sspace-A-%d", testConfig.Prefix, rand.Int31())
				Expect(cf.Cf("create-space", spaceNameA, "-o", TestSetup.TestSpace.OrganizationName()).Wait(Timeout_Push)).To(gexec.Exit(0))

				spaceNameB = fmt.Sprintf("%sspace-B-%d", testConfig.Prefix, rand.Int31())
				Expect(cf.Cf("create-space", spaceNameB, "-o", TestSetup.TestSpace.OrganizationName()).Wait(Timeout_Push)).To(gexec.Exit(0))

				Expect(cf.Cf("target", "-o", TestSetup.TestSpace.OrganizationName(), "-s", spaceNameA).Wait(Timeout_Push)).To(gexec.Exit(0))
				pushProxy(appA)

				Expect(cf.Cf("target", "-o", TestSetup.TestSpace.OrganizationName(), "-s", spaceNameB).Wait(Timeout_Push)).To(gexec.Exit(0))
				pushProxy(appB)
			})
		})

		Describe("authenticating as a user", func() {
			BeforeEach(func() {
				workflowhelpers.AsUser(TestSetup.AdminUserContext(), Timeout_Push, func() {
					createUserSession := cf.Cf("create-user", "space-developer", "password")
					Expect(createUserSession.Wait(Timeout_Push)).To(gexec.Exit(0))

					Expect(cf.Cf("set-space-role", "space-developer", TestSetup.TestSpace.OrganizationName(), spaceNameA, "SpaceDeveloper").Wait(Timeout_Push)).To(gexec.Exit(0))
					Expect(cf.Cf("set-space-role", "space-developer", TestSetup.TestSpace.OrganizationName(), spaceNameB, "SpaceDeveloper").Wait(Timeout_Push)).To(gexec.Exit(0))
				})
			})

			It("can create, list, and delete network policies in spaces they have access to", func() {
				var spaceDevUserToken string
				var appAGUID, appBGUID string

				By("logging in and getting the space developer user token")
				Expect(cf.Cf("auth", "space-developer", "password").Wait(Timeout_Push)).To(gexec.Exit(0))
				session := cf.Cf("oauth-token")
				Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))

				spaceDevUserToken = strings.TrimSpace(string(session.Out.Contents()))

				By("getting the app guids")
				Expect(cf.Cf("target", "-o", TestSetup.TestSpace.OrganizationName(), "-s", spaceNameA).Wait(Timeout_Push)).To(gexec.Exit(0))
				session = cf.Cf("app", appA, "--guid")
				Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))
				appAGUID = strings.TrimSpace(string(session.Out.Contents()))

				Expect(cf.Cf("target", "-o", TestSetup.TestSpace.OrganizationName(), "-s", spaceNameB).Wait(Timeout_Push)).To(gexec.Exit(0))
				session = cf.Cf("app", appB, "--guid")
				Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))
				appBGUID = strings.TrimSpace(string(session.Out.Contents()))

				By("creating a policy")
				err := policyClient.AddPoliciesV0(spaceDevUserToken, []policy_client.PolicyV0{
					{
						Source: policy_client.SourceV0{
							ID: appAGUID,
						},
						Destination: policy_client.DestinationV0{
							ID:       appBGUID,
							Port:     1234,
							Protocol: "tcp",
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				By("listing policies")
				expectedPolicies := []policy_client.PolicyV0{
					{
						Source: policy_client.SourceV0{
							ID: appAGUID,
						},
						Destination: policy_client.DestinationV0{
							ID:       appBGUID,
							Port:     1234,
							Protocol: "tcp",
						},
					},
				}
				policies, err := policyClient.GetPoliciesV0(spaceDevUserToken)
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(Equal(expectedPolicies))

				By("deleting the policy")
				err = policyClient.DeletePoliciesV0(spaceDevUserToken, []policy_client.PolicyV0{
					{
						Source: policy_client.SourceV0{
							ID: appAGUID,
						},
						Destination: policy_client.DestinationV0{
							ID:       appBGUID,
							Port:     1234,
							Protocol: "tcp",
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())

			})
		})

		Describe("authenticating as a UAA client", func() {
			var spaceClient string
			var uaaAPI *uaa.API

			BeforeEach(func() {
				var err error

				uaaAPI, err = uaa.New(getUAABaseURL(), uaa.WithClientCredentials("admin", testConfig.AdminSecret, uaa.OpaqueToken), uaa.WithSkipSSLValidation(true))
				Expect(err).NotTo(HaveOccurred())

				users, _, err := uaaAPI.ListUsers(fmt.Sprintf(`userName eq %q`, "space-developer"), "", "", "", 0, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(users).To(HaveLen(1))
				user := users[0]

				group := ensureGroupExists("network.write", uaaAPI)

				err = uaaAPI.AddGroupMember(group.ID, user.ID, "", "")

				if err != nil {
					Expect(err).To(BeAssignableToTypeOf(uaa.RequestError{}))
					Expect(err.(uaa.RequestError).ErrorResponse).To(ContainSubstring("already_exists"))
				}

				spaceClient = fmt.Sprintf("space-client-%d", rand.Int31())

				_, err = uaaAPI.CreateClient(uaa.Client{
					ClientID:             spaceClient,
					ClientSecret:         "password",
					DisplayName:          "space-client",
					Authorities:          []string{"network.write", "cloud_controller.read"},
					AuthorizedGrantTypes: []string{"client_credentials"},
					AccessTokenValidity:  600,
				})
				Expect(err).NotTo(HaveOccurred())

				workflowhelpers.AsUser(TestSetup.AdminUserContext(), Timeout_Push, func() {
					Expect(cf.Cf("target", "-o", TestSetup.TestSpace.OrganizationName()).Wait(Timeout_Push)).To(gexec.Exit(0))
					orgGuid, err := cfCLI.OrgGuid(TestSetup.TestSpace.OrganizationName())
					Expect(err).NotTo(HaveOccurred())

					spaceAGuid, err := cfCLI.SpaceGuid(spaceNameA)
					Expect(err).NotTo(HaveOccurred(), spaceAGuid)

					spaceBGuid, err := cfCLI.SpaceGuid(spaceNameB)
					Expect(err).NotTo(HaveOccurred(), spaceBGuid)

					Expect(cf.Cf("curl", "-X", "PUT", fmt.Sprintf("/v2/organizations/%s/users/%s", orgGuid, spaceClient)).Wait()).To(gexec.Exit(0))
					Expect(cf.Cf("curl", "-X", "PUT", fmt.Sprintf("/v2/spaces/%s/developers/%s", spaceAGuid, spaceClient)).Wait(Timeout_Push)).To(gexec.Exit(0))
					Expect(cf.Cf("curl", "-X", "PUT", fmt.Sprintf("/v2/spaces/%s/developers/%s", spaceBGuid, spaceClient)).Wait(Timeout_Push)).To(gexec.Exit(0))
				})
			})

			AfterEach(func() {
				uaaAPI.DeleteClient(spaceClient)
			})

			It("can create, list, and delete network policies in spaces they have access to", func() {
				var spaceDevUserToken string

				By("logging in and getting the space developer user token")
				Expect(cf.Cf("auth", spaceClient, "password", "--client-credentials").Wait(Timeout_Push)).To(gexec.Exit(0))
				session := cf.Cf("oauth-token")
				Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))

				spaceDevUserToken = strings.TrimSpace(string(session.Out.Contents()))

				var appAGUID, appBGUID string
				By("getting the app guids")
				Expect(cf.Cf("target", "-o", TestSetup.TestSpace.OrganizationName(), "-s", spaceNameA).Wait(Timeout_Push)).To(gexec.Exit(0))
				session = cf.Cf("app", appA, "--guid")
				Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))
				appAGUID = strings.TrimSpace(string(session.Out.Contents()))

				Expect(cf.Cf("target", "-o", TestSetup.TestSpace.OrganizationName(), "-s", spaceNameB).Wait(Timeout_Push)).To(gexec.Exit(0))
				session = cf.Cf("app", appB, "--guid")
				Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))
				appBGUID = strings.TrimSpace(string(session.Out.Contents()))

				By("creating a policy")
				err := policyClient.AddPoliciesV0(spaceDevUserToken, []policy_client.PolicyV0{
					{
						Source: policy_client.SourceV0{
							ID: appAGUID,
						},
						Destination: policy_client.DestinationV0{
							ID:       appBGUID,
							Port:     1234,
							Protocol: "tcp",
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				By("listing policies")
				expectedPolicies := []policy_client.PolicyV0{
					{
						Source: policy_client.SourceV0{
							ID: appAGUID,
						},
						Destination: policy_client.DestinationV0{
							ID:       appBGUID,
							Port:     1234,
							Protocol: "tcp",
						},
					},
				}
				policies, err := policyClient.GetPoliciesV0(spaceDevUserToken)
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(Equal(expectedPolicies))

				By("deleting the policy")
				err = policyClient.DeletePoliciesV0(spaceDevUserToken, []policy_client.PolicyV0{
					{
						Source: policy_client.SourceV0{
							ID: appAGUID,
						},
						Destination: policy_client.DestinationV0{
							ID:       appBGUID,
							Port:     1234,
							Protocol: "tcp",
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())
			})

		})
	})
})

func ensureGroupExists(name string, uaaAPI *uaa.API) uaa.Group {
	_, err := uaaAPI.CreateGroup(uaa.Group{DisplayName: name})
	if err != nil {
		Expect(err).To(BeAssignableToTypeOf(uaa.RequestError{}))
		Expect(err.(uaa.RequestError).ErrorResponse).To(ContainSubstring("already_exists"))
	}

	groups, _, err := uaaAPI.ListGroups(fmt.Sprintf(`displayName eq %q`, name), "", "", "", 0, 0)
	Expect(err).NotTo(HaveOccurred())
	Expect(groups).To(HaveLen(1))

	return groups[0]
}
