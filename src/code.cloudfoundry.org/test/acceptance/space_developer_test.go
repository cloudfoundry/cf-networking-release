package acceptance_test

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"code.cloudfoundry.org/lager/v3/lagertest"
	"code.cloudfoundry.org/policy_client"
	uaa "github.com/cloudfoundry-community/go-uaa"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const spaceClientID = "45733a09-a1c2-4990-b0ee-3e0428552c05"

var _ = Describe("space developer policy configuration", func() {
	var (
		appA       string
		appB       string
		orgName    string
		spaceNameA string
		spaceNameB string
		prefix     string

		policyClient *policy_client.ExternalClient

		uaaAPI *uaa.API
	)

	BeforeEach(func() {
		var err error
		uaaAPI, err = uaa.New(getUAABaseURL(), uaa.WithClientCredentials("admin", testConfig.AdminSecret, uaa.OpaqueToken), uaa.WithSkipSSLValidation(true))
		Expect(err).ToNot(HaveOccurred())

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

		prefix = testConfig.Prefix
		appA = fmt.Sprintf("appA-%d", randomGenerator.Int31())
		appB = fmt.Sprintf("appB-%d", randomGenerator.Int31())

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

		user := ensureUserExists("space-developer", "password", uaaAPI)
		group := ensureGroupExists("network.write", uaaAPI)

		err = uaaAPI.AddGroupMember(group.ID, user.ID, "", "")
		if err != nil {
			Expect(err).To(BeAssignableToTypeOf(uaa.RequestError{}))
			Expect(err.(uaa.RequestError).ErrorResponse).To(ContainSubstring("already_exists"))
		}

		Expect(cf.Cf("set-space-role", "space-developer", orgName, spaceNameA, "SpaceDeveloper").Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("set-space-role", "space-developer", orgName, spaceNameB, "SpaceDeveloper").Wait(Timeout_Push)).To(gexec.Exit(0))

		_, err = uaaAPI.CreateClient(uaa.Client{
			ClientID:             spaceClientID,
			ClientSecret:         "password",
			DisplayName:          "space-client",
			Authorities:          []string{"network.write", "cloud_controller.read"},
			AuthorizedGrantTypes: []string{"client_credentials"},
			AccessTokenValidity:  600,
		})
		Expect(err).NotTo(HaveOccurred())

		orgGuid, err := cfCLI.OrgGuid(orgName)
		Expect(err).NotTo(HaveOccurred())

		spaceAGuid, err := cfCLI.SpaceGuid(spaceNameA)
		Expect(err).NotTo(HaveOccurred())

		spaceBGuid, err := cfCLI.SpaceGuid(spaceNameB)
		Expect(err).NotTo(HaveOccurred())

		type organization struct {
			Data struct {
				Guid string `json:"guid,omitempty"`
			} `json:"data,omitempty"`
		}
		type space struct {
			Data struct {
				Guid string `json:"guid,omitempty"`
			} `json:"data,omitempty"`
		}
		type role struct {
			Type          string `json:"type"`
			Relationships struct {
				User struct {
					Data struct {
						Guid string `json:"guid"`
					} `json:"data"`
				} `json:"user"`
				Organization *organization `json:"organization,omitempty"`
				Space        *space        `json:"space,omitempty"`
			} `json:"relationships"`
		}

		r := new(role)
		r.Type = "organization_user"
		r.Relationships.User.Data.Guid = spaceClientID
		r.Relationships.Organization = &organization{}
		r.Relationships.Organization.Data.Guid = orgGuid

		body, err := json.Marshal(r)
		Expect(err).NotTo(HaveOccurred())

		Expect(cf.Cf("curl", "-X", "POST", "/v3/roles", "-d", string(body)).Wait()).To(gexec.Exit(0))

		r1 := new(role)
		r1.Type = "space_developer"
		r1.Relationships.User.Data.Guid = spaceClientID
		r1.Relationships.Space = &space{}
		r1.Relationships.Space.Data.Guid = spaceAGuid
		body, err = json.Marshal(r1)
		Expect(err).NotTo(HaveOccurred())
		Expect(cf.Cf("curl", "-X", "POST", "/v3/roles", "-d", string(body)).Wait()).To(gexec.Exit(0))

		r1 = new(role)
		r1.Type = "space_developer"
		r1.Relationships.User.Data.Guid = spaceClientID
		r1.Relationships.Space = &space{}
		r1.Relationships.Space.Data.Guid = spaceBGuid
		body, err = json.Marshal(r1)
		Expect(err).NotTo(HaveOccurred())
		Expect(cf.Cf("curl", "-X", "POST", "/v3/roles", "-d", string(body)).Wait()).To(gexec.Exit(0))

	})

	AfterEach(func() {
		By("logging in as admin and deleting the org", func() {
			Expect(cf.Cf("logout").Wait(Timeout_Push)).To(gexec.Exit(0))
			Expect(cf.Cf("auth", config.AdminUser, config.AdminPassword).Wait(Timeout_Push)).To(gexec.Exit(0))

			uaaAPI.DeleteClient(spaceClientID)

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
			})

			By("listing policies", func() {
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
			})

			By("deleting the policy", func() {
				err := policyClient.DeletePoliciesV0(spaceDevUserToken, []policy_client.PolicyV0{
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
		},
			Entry("as a user", []string{"space-developer", "password"}),
			Entry("as a service account", []string{spaceClientID, "password", "--client-credentials"}),
		)
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

func ensureUserExists(username string, password string, uaaAPI *uaa.API) uaa.User {
	createUserSession := cf.Cf("create-user", username, password)
	Expect(createUserSession.Wait(Timeout_Push)).To(gexec.Exit(0))

	users, _, err := uaaAPI.ListUsers(fmt.Sprintf(`userName eq %q`, username), "", "", "", 0, 0)
	Expect(err).NotTo(HaveOccurred())
	Expect(users).To(HaveLen(1))

	return users[0]
}
