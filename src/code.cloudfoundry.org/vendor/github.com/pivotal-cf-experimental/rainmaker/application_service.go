package rainmaker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pivotal-cf-experimental/rainmaker/internal/documents"
	"github.com/pivotal-cf-experimental/rainmaker/internal/network"
)

type ApplicationsService struct {
	config Config
}

func NewApplicationsService(config Config) ApplicationsService {
	return ApplicationsService{
		config: config,
	}
}

func (service ApplicationsService) Create(application Application, token string) (Application, error) {
	resp, err := newNetworkClient(service.config).MakeRequest(network.Request{
		Method: "POST",
		Path:   "/v2/apps",
		Body: network.NewJSONRequestBody(documents.CreateApplicationRequest{
			Name:      application.Name,
			SpaceGUID: application.SpaceGUID,
			Diego:     application.Diego,
		}),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusCreated},
	})
	if err != nil {
		return Application{}, translateError(err)
	}

	var response documents.ApplicationResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		panic(err)
	}

	return newApplicationFromResponse(service.config, response), nil
}

func (service ApplicationsService) List(token string) (ApplicationsList, error) {
	list := NewApplicationsList(service.config, newRequestPlan("/v2/apps", url.Values{}))
	err := list.Fetch(token)

	return list, err
}

func (service ApplicationsService) Get(guid, token string) (Application, error) {
	resp, err := newNetworkClient(service.config).MakeRequest(network.Request{
		Method:                "GET",
		Path:                  fmt.Sprintf("/v2/apps/%s", guid),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return Application{}, translateError(err)
	}

	var response documents.ApplicationResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return Application{}, translateError(err)
	}

	return newApplicationFromResponse(service.config, response), nil
}

func (service ApplicationsService) Delete(guid, token string) error {
	_, err := newNetworkClient(service.config).MakeRequest(network.Request{
		Method:                "DELETE",
		Path:                  fmt.Sprintf("/v2/apps/%s", guid),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusNoContent},
	})
	if err != nil {
		return translateError(err)
	}

	return nil
}
