package rainmaker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pivotal-cf-experimental/rainmaker/internal/documents"
	"github.com/pivotal-cf-experimental/rainmaker/internal/network"
)

type OrganizationsService struct {
	config Config
}

func NewOrganizationsService(config Config) OrganizationsService {
	return OrganizationsService{
		config: config,
	}
}

func (service OrganizationsService) Create(name string, token string) (Organization, error) {
	resp, err := newNetworkClient(service.config).MakeRequest(network.Request{
		Method: "POST",
		Path:   "/v2/organizations",
		Body: network.NewJSONRequestBody(documents.CreateOrganizationRequest{
			Name: name,
		}),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusCreated},
	})
	if err != nil {
		return Organization{}, err
	}

	var response documents.OrganizationResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		panic(err)
	}

	return newOrganizationFromResponse(service.config, response), nil
}

func (service OrganizationsService) List(token string) (OrganizationsList, error) {
	list := NewOrganizationsList(service.config, newRequestPlan("/v2/organizations", url.Values{}))
	err := list.Fetch(token)

	return list, err
}

func (service OrganizationsService) Get(guid, token string) (Organization, error) {
	resp, err := newNetworkClient(service.config).MakeRequest(network.Request{
		Method:                "GET",
		Path:                  fmt.Sprintf("/v2/organizations/%s", guid),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return Organization{}, translateError(err)
	}

	var response documents.OrganizationResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return Organization{}, translateError(err)
	}

	return newOrganizationFromResponse(service.config, response), nil
}

func (service OrganizationsService) Delete(guid, token string) error {
	_, err := newNetworkClient(service.config).MakeRequest(network.Request{
		Method:                "DELETE",
		Path:                  fmt.Sprintf("/v2/organizations/%s", guid),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusNoContent},
	})
	if err != nil {
		return translateError(err)
	}

	return nil
}

func (service OrganizationsService) Update(org Organization, token string) (Organization, error) {
	resp, err := newNetworkClient(service.config).MakeRequest(network.Request{
		Method: "PUT",
		Path:   fmt.Sprintf("/v2/organizations/%s", org.GUID),
		Body: network.NewJSONRequestBody(documents.UpdateOrganizationRequest{
			Name:                org.Name,
			Status:              org.Status,
			QuotaDefinitionGUID: org.QuotaDefinitionGUID,
		}),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusCreated},
	})
	if err != nil {
		return Organization{}, translateError(err)
	}

	var response documents.OrganizationResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return Organization{}, translateError(err)
	}

	return newOrganizationFromResponse(service.config, response), nil
}

func (service OrganizationsService) ListSpaces(guid, token string) (SpacesList, error) {
	list := NewSpacesList(service.config, newRequestPlan("/v2/organizations/"+guid+"/spaces", url.Values{}))
	err := list.Fetch(token)
	if err != nil {
		return SpacesList{}, translateError(err)
	}

	return list, nil
}

func (service OrganizationsService) ListUsers(guid, token string) (UsersList, error) {
	list := NewUsersList(service.config, newRequestPlan("/v2/organizations/"+guid+"/users", url.Values{}))
	err := list.Fetch(token)
	if err != nil {
		return UsersList{}, translateError(err)
	}

	return list, nil
}

func (service OrganizationsService) ListBillingManagers(guid, token string) (UsersList, error) {
	list := NewUsersList(service.config, newRequestPlan("/v2/organizations/"+guid+"/billing_managers", url.Values{}))
	err := list.Fetch(token)

	return list, err
}

func (service OrganizationsService) ListAuditors(guid, token string) (UsersList, error) {
	list := NewUsersList(service.config, newRequestPlan("/v2/organizations/"+guid+"/auditors", url.Values{}))
	err := list.Fetch(token)

	return list, err
}

func (service OrganizationsService) ListManagers(guid, token string) (UsersList, error) {
	list := NewUsersList(service.config, newRequestPlan("/v2/organizations/"+guid+"/managers", url.Values{}))
	err := list.Fetch(token)

	return list, err
}
