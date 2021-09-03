package rainmaker

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pivotal-cf-experimental/rainmaker/internal/documents"
	"github.com/pivotal-cf-experimental/rainmaker/internal/network"
)

type ServiceInstancesService struct {
	config Config
}

func NewServiceInstancesService(config Config) ServiceInstancesService {
	return ServiceInstancesService{
		config: config,
	}
}

func (service *ServiceInstancesService) Create(name, planGUID, spaceGUID, token string) (ServiceInstance, error) {
	resp, err := newNetworkClient(service.config).MakeRequest(network.Request{
		Method: "POST",
		Path:   "/v2/service_instances",
		Body: network.NewJSONRequestBody(documents.CreateServiceInstanceRequest{
			Name:      name,
			PlanGUID:  planGUID,
			SpaceGUID: spaceGUID,
		}),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusCreated},
	})
	if err != nil {
		return ServiceInstance{}, err
	}

	var response documents.ServiceInstanceResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		panic(err)
	}

	return newServiceInstanceFromResponse(service.config, response), nil
}

func (service *ServiceInstancesService) Get(guid, token string) (ServiceInstance, error) {
	resp, err := newNetworkClient(service.config).MakeRequest(network.Request{
		Method:                "GET",
		Path:                  fmt.Sprintf("/v2/service_instances/%s", guid),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return ServiceInstance{}, err
	}

	var response documents.ServiceInstanceResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		panic(err)
	}
	return newServiceInstanceFromResponse(service.config, response), nil
}
