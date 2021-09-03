package rainmaker

import (
	"encoding/json"
	"net/http"

	"github.com/pivotal-cf-experimental/rainmaker/internal/documents"
	"github.com/pivotal-cf-experimental/rainmaker/internal/network"
)

type UsersService struct {
	config Config
	user   User
}

func NewUsersService(config Config) UsersService {
	return UsersService{
		config: config,
	}
}

func (service UsersService) Create(guid, token string) (User, error) {
	resp, err := newNetworkClient(service.config).MakeRequest(network.Request{
		Method: "POST",
		Path:   "/v2/users",
		Body: network.NewJSONRequestBody(documents.CreateUserRequest{
			GUID: guid,
		}),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusCreated},
	})
	if err != nil {
		return User{}, err
	}

	var document documents.UserResponse
	err = json.Unmarshal(resp.Body, &document)
	if err != nil {
		panic(err)
	}

	return newUserFromResponse(service.config, document), nil
}

func (service UsersService) Get(guid, token string) (User, error) {
	resp, err := newNetworkClient(service.config).MakeRequest(network.Request{
		Method:                "GET",
		Path:                  "/v2/users/" + guid,
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return User{}, err
	}

	var response documents.UserResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return User{}, translateError(err)
	}

	return newUserFromResponse(service.config, response), nil
}
