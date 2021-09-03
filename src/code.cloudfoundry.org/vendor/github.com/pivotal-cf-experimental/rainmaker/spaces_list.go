package rainmaker

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/pivotal-cf-experimental/rainmaker/internal/documents"
	"github.com/pivotal-cf-experimental/rainmaker/internal/network"
)

type SpacesList struct {
	config       Config
	plan         requestPlan
	TotalResults int
	TotalPages   int
	NextURL      string
	PrevURL      string
	Spaces       []Space
}

func NewSpacesList(config Config, plan requestPlan) SpacesList {
	return SpacesList{
		config: config,
		plan:   plan,
	}
}

func (list SpacesList) Create(space Space, token string) (Space, error) {
	var document documents.SpaceResponse
	resp, err := newNetworkClient(list.config).MakeRequest(network.Request{
		Method:        "POST",
		Path:          list.plan.Path,
		Authorization: network.NewTokenAuthorization(token),
		Body:          network.NewJSONRequestBody(space),
		AcceptableStatusCodes: []int{http.StatusCreated},
	})
	if err != nil {
		return Space{}, err
	}

	err = json.Unmarshal(resp.Body, &document)
	if err != nil {
		panic(err)
	}

	return newSpaceFromResponse(list.config, document), nil
}

func (list SpacesList) Next(token string) (SpacesList, error) {
	nextURL, err := url.Parse("http://example.com" + list.NextURL)
	if err != nil {
		return SpacesList{}, err
	}

	nextList := NewSpacesList(list.config, newRequestPlan(nextURL.Path, nextURL.Query()))
	err = nextList.Fetch(token)

	return nextList, err
}

func (list SpacesList) Prev(token string) (SpacesList, error) {
	prevURL, err := url.Parse("http://example.com" + list.PrevURL)
	if err != nil {
		return SpacesList{}, err
	}

	prevList := NewSpacesList(list.config, newRequestPlan(prevURL.Path, prevURL.Query()))
	err = prevList.Fetch(token)

	return prevList, err
}

func (list SpacesList) HasNextPage() bool {
	return list.NextURL != ""
}

func (list SpacesList) HasPrevPage() bool {
	return list.PrevURL != ""
}

func (list *SpacesList) Fetch(token string) error {
	u := url.URL{
		Path:     list.plan.Path,
		RawQuery: list.plan.Query.Encode(),
	}

	resp, err := newNetworkClient(list.config).MakeRequest(network.Request{
		Method:                "GET",
		Path:                  u.String(),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return err
	}

	var response documents.SpacesListResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		panic(err)
	}

	updatedList := newSpacesListFromResponse(list.config, list.plan, response)
	list.TotalResults = updatedList.TotalResults
	list.TotalPages = updatedList.TotalPages
	list.NextURL = updatedList.NextURL
	list.PrevURL = updatedList.PrevURL
	list.Spaces = updatedList.Spaces

	return nil
}

func newSpacesListFromResponse(config Config, plan requestPlan, response documents.SpacesListResponse) SpacesList {
	list := NewSpacesList(config, plan)
	list.TotalResults = response.TotalResults
	list.TotalPages = response.TotalPages
	list.PrevURL = response.PrevURL
	list.NextURL = response.NextURL
	list.Spaces = make([]Space, 0)

	for _, spaceResponse := range response.Resources {
		list.Spaces = append(list.Spaces, newSpaceFromResponse(config, spaceResponse))
	}

	return list
}
