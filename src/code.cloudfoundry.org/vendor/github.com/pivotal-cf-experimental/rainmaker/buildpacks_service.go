package rainmaker

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pivotal-cf-experimental/rainmaker/internal/documents"
	"github.com/pivotal-cf-experimental/rainmaker/internal/network"
)

type BuildpacksService struct {
	config Config
}

type CreateBuildpackOptions struct {
	Position *int
	Enabled  *bool
	Locked   *bool
	Filename *string
}

func NewBuildpacksService(config Config) BuildpacksService {
	return BuildpacksService{
		config: config,
	}
}

func (b BuildpacksService) Create(name string, token string, options *CreateBuildpackOptions) (Buildpack, error) {
	requestBody := documents.CreateBuildpackRequest{
		Name: name,
	}

	if options != nil {
		requestBody.Position = options.Position
		requestBody.Enabled = options.Enabled
		requestBody.Locked = options.Locked
		requestBody.Filename = options.Filename
	}

	resp, err := newNetworkClient(b.config).MakeRequest(network.Request{
		Method:                "POST",
		Path:                  "/v2/buildpacks",
		Body:                  network.NewJSONRequestBody(requestBody),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusCreated},
	})
	if err != nil {
		panic(err)
	}

	var response documents.BuildpackResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		panic(err)
	}

	return newBuildpackFromResponse(b.config, response), nil
}

func (b BuildpacksService) Get(guid string, token string) (Buildpack, error) {
	resp, err := newNetworkClient(b.config).MakeRequest(network.Request{
		Method:                "GET",
		Path:                  fmt.Sprintf("/v2/buildpacks/%s", guid),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return Buildpack{}, translateError(err)
	}

	var response documents.BuildpackResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		panic(err)
	}

	return newBuildpackFromResponse(b.config, response), nil
}

func (b BuildpacksService) Delete(guid, token string) error {
	_, err := newNetworkClient(b.config).MakeRequest(network.Request{
		Method:                "DELETE",
		Path:                  fmt.Sprintf("/v2/buildpacks/%s", guid),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusNoContent},
	})
	if err != nil {
		return err
	}

	return nil
}

func (b BuildpacksService) Update(buildpack Buildpack, token string) (Buildpack, error) {
	requestBody := documents.UpdateBuildpackRequest{
		Position: &buildpack.Position,
		Enabled:  &buildpack.Enabled,
		Locked:   &buildpack.Locked,
		Filename: &buildpack.Filename,
		Name:     &buildpack.Name,
	}

	resp, err := newNetworkClient(b.config).MakeRequest(network.Request{
		Method:                "PUT",
		Path:                  fmt.Sprintf("/v2/buildpacks/%s", buildpack.GUID),
		Body:                  network.NewJSONRequestBody(requestBody),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusCreated},
	})
	if err != nil {
		return Buildpack{}, translateError(err)
	}

	var response documents.BuildpackResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return Buildpack{}, translateError(err)
	}

	return newBuildpackFromResponse(b.config, response), nil
}
