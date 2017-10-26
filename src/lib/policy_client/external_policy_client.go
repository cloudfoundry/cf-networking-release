package policy_client

import (
	"fmt"
	"net/http"
	"policy-server/api/api_v0"
	"strings"

	"policy-server/api"

	"code.cloudfoundry.org/cf-networking-helpers/json_client"
	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/external_policy_client.go --fake-name ExternalPolicyClient . ExternalPolicyClient
type ExternalPolicyClient interface {
	GetAPIVersion() (int, error)
	GetPolicies(token string) ([]api.Policy, error)
	GetPoliciesByID(token string, ids ...string) ([]api.Policy, error)
	GetPoliciesV0(token string) ([]api_v0.Policy, error)
	GetPoliciesV0ByID(token string, ids ...string) ([]api_v0.Policy, error)
	DeletePolicies(token string, policies []api.Policy) error
	DeletePoliciesV0(token string, policies []api_v0.Policy) error
	AddPolicies(token string, policies []api.Policy) error
	AddPoliciesV0(token string, policies []api_v0.Policy) error
}

type ExternalClient struct {
	JsonClient json_client.JsonClient
	Chunker    Chunker
}

func NewExternal(logger lager.Logger, httpClient json_client.HttpClient, baseURL string) *ExternalClient {
	return &ExternalClient{
		JsonClient: json_client.New(logger, httpClient, baseURL),
		Chunker:    &SimpleChunker{ChunkSize: DefaultMaxPolicies},
	}
}

func (c *ExternalClient) GetAPIVersion() (int, error) {
	var versionResp struct {
		Links struct {
			NetworkPolicy struct {
				Href string `json:"href"`
			} `json:"network_policy"`
			NetworkPolicyV0 struct {
				Href string `json:"href"`
			} `json:"network_policy_v0"`
			NetworkPolicyV1 struct {
				Href string `json:"href"`
			} `json:"network_policy_v1"`
		} `json:"links"`
	}

	err := c.JsonClient.Do("GET", "/", nil, &versionResp, "")
	if err != nil {
		return 0, parseHttpError(err)
	}

	// This commit allows us to simplify how we determine policy server api version
	// https://github.com/cloudfoundry/cloud_controller_ng/commit/ea8277b806c7b3290682ce22df7566e44136e55e#diff-313d9acd8a9482431178d1d5ba7e7f82
	if versionResp.Links.NetworkPolicy.Href != "" {
		return 0, nil
	}

	if versionResp.Links.NetworkPolicyV1.Href != "" {
		return 1, nil
	}

	return -1, nil
}

func (c *ExternalClient) GetPolicies(token string) ([]api.Policy, error) {
	var policies struct {
		Policies []api.Policy `json:"policies"`
	}
	err := c.JsonClient.Do("GET", "/networking/v1/external/policies", nil, &policies, token)
	if err != nil {
		return nil, parseHttpError(err)
	}
	return policies.Policies, nil
}

func (c *ExternalClient) GetPoliciesByID(token string, ids ...string) ([]api.Policy, error) {
	var policies struct {
		Policies []api.Policy `json:"policies"`
	}
	route := "/networking/v1/external/policies?id=" + strings.Join(ids, ",")
	err := c.JsonClient.Do("GET", route, nil, &policies, token)
	if err != nil {
		return nil, parseHttpError(err)
	}
	return policies.Policies, nil
}

func (c *ExternalClient) GetPoliciesV0(token string) ([]api_v0.Policy, error) {
	var policies struct {
		Policies []api_v0.Policy `json:"policies"`
	}
	err := c.JsonClient.Do("GET", "/networking/v0/external/policies", nil, &policies, token)
	if err != nil {
		return nil, parseHttpError(err)
	}
	return policies.Policies, nil
}

func (c *ExternalClient) GetPoliciesV0ByID(token string, ids ...string) ([]api_v0.Policy, error) {
	var policies struct {
		Policies []api_v0.Policy `json:"policies"`
	}
	route := "/networking/v0/external/policies?id=" + strings.Join(ids, ",")
	err := c.JsonClient.Do("GET", route, nil, &policies, token)
	if err != nil {
		return nil, parseHttpError(err)
	}
	return policies.Policies, nil
}

func (c *ExternalClient) AddPolicies(token string, policies []api.Policy) error {
	reqPolicies := map[string][]api.Policy{
		"policies": policies,
	}

	err := c.JsonClient.Do("POST", "/networking/v1/external/policies", reqPolicies, nil, token)
	if err != nil {
		return parseHttpError(err)
	}

	return nil
}

func (c *ExternalClient) AddPoliciesV0(token string, policies []api_v0.Policy) error {
	chunks := c.Chunker.Chunk(policies)
	for _, chunk := range chunks {
		reqPolicies := map[string][]api_v0.Policy{
			"policies": chunk,
		}
		err := c.JsonClient.Do("POST", "/networking/v0/external/policies", reqPolicies, nil, token)
		if err != nil {
			return parseHttpError(err)
		}
	}
	return nil
}

func (c *ExternalClient) DeletePolicies(token string, policies []api.Policy) error {
	reqPolicies := map[string][]api.Policy{
		"policies": policies,
	}

	err := c.JsonClient.Do("POST", "/networking/v1/external/policies/delete", reqPolicies, nil, token)
	if err != nil {
		return parseHttpError(err)
	}

	return nil
}

func (c *ExternalClient) DeletePoliciesV0(token string, policies []api_v0.Policy) error {
	chunks := c.Chunker.Chunk(policies)
	for _, chunk := range chunks {
		reqPolicies := map[string][]api_v0.Policy{
			"policies": chunk,
		}
		err := c.JsonClient.Do("POST", "/networking/v0/external/policies/delete", reqPolicies, nil, token)
		if err != nil {
			return parseHttpError(err)
		}
	}
	return nil
}

// Check if error is bad status code and parse out the JSON body
func parseHttpError(err error) error {
	httpErr, ok := err.(*json_client.HttpResponseCodeError)
	if ok {
		return fmt.Errorf("%d %s: %s", httpErr.StatusCode,
			http.StatusText(httpErr.StatusCode),
			httpErr.Message,
		)
	}
	return err
}
