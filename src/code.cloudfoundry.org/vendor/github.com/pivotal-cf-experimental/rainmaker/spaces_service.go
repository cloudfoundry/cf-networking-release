package rainmaker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pivotal-cf-experimental/rainmaker/internal/documents"
	"github.com/pivotal-cf-experimental/rainmaker/internal/network"
)

type SpacesService struct {
	config Config
}

func NewSpacesService(config Config) SpacesService {
	return SpacesService{
		config: config,
	}
}

func (service SpacesService) Create(name, orgGUID, token string) (Space, error) {
	resp, err := newNetworkClient(service.config).MakeRequest(network.Request{
		Method: "POST",
		Path:   "/v2/spaces",
		Body: network.NewJSONRequestBody(documents.CreateSpaceRequest{
			Name:             name,
			OrganizationGUID: orgGUID,
		}),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusCreated},
	})
	if err != nil {
		return Space{}, err
	}

	var response documents.SpaceResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		panic(err)
	}

	return newSpaceFromResponse(service.config, response), nil
}

func (service SpacesService) List(token string) (SpacesList, error) {
	list := NewSpacesList(service.config, newRequestPlan("/v2/spaces", url.Values{}))
	err := list.Fetch(token)

	return list, err
}

func (service SpacesService) Get(guid, token string) (Space, error) {
	resp, err := newNetworkClient(service.config).MakeRequest(network.Request{
		Method:                "GET",
		Path:                  fmt.Sprintf("/v2/spaces/%s", guid),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return Space{}, translateError(err)
	}

	var response documents.SpaceResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return Space{}, translateError(err)
	}

	return newSpaceFromResponse(service.config, response), nil
}

func (service SpacesService) Delete(guid, token string) error {
	_, err := newNetworkClient(service.config).MakeRequest(network.Request{
		Method:                "DELETE",
		Path:                  fmt.Sprintf("/v2/spaces/%s", guid),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusNoContent},
	})
	if err != nil {
		return translateError(err)
	}

	return nil
}

func (service SpacesService) ListUsers(guid, token string) (UsersList, error) {
	query := url.Values{}
	query.Set("q", fmt.Sprintf("space_guid:%s", guid))

	list := NewUsersList(service.config, newRequestPlan("/v2/users", query))
	err := list.Fetch(token)

	return list, err
}
