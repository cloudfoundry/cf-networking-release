package rainmaker

import (
	"encoding/json"
	"net/http"

	"github.com/pivotal-cf-experimental/rainmaker/internal/documents"
	"github.com/pivotal-cf-experimental/rainmaker/internal/network"
)

type OrganizationsList struct {
	config Config
	plan   requestPlan
	Page

	Organizations []Organization
}

func NewOrganizationsList(config Config, plan requestPlan) OrganizationsList {
	return OrganizationsList{
		config: config,
		plan:   plan,
		Page:   NewPage(config, plan),
	}
}

func (list OrganizationsList) Create(org Organization, token string) (Organization, error) {
	var document documents.OrganizationResponse
	resp, err := newNetworkClient(list.config).MakeRequest(network.Request{
		Method:        "POST",
		Path:          list.plan.Path,
		Authorization: network.NewTokenAuthorization(token),
		Body:          network.NewJSONRequestBody(org),
		AcceptableStatusCodes: []int{http.StatusCreated},
	})
	if err != nil {
		return Organization{}, err
	}

	err = json.Unmarshal(resp.Body, &document)
	if err != nil {
		panic(err)
	}

	return newOrganizationFromResponse(list.config, document), nil
}

func (list OrganizationsList) Next(token string) (OrganizationsList, error) {
	nextPage, err := list.Page.Next(token)
	if err != nil {
		return OrganizationsList{}, err
	}

	nextList := newOrganizationsListFromPage(list.config, nextPage.plan, nextPage)
	err = nextList.Fetch(token)

	return nextList, err
}

func (list OrganizationsList) Prev(token string) (OrganizationsList, error) {
	prevPage, err := list.Page.Prev(token)
	if err != nil {
		return OrganizationsList{}, err
	}

	prevList := newOrganizationsListFromPage(list.config, prevPage.plan, prevPage)
	err = prevList.Fetch(token)

	return prevList, err
}

func (list *OrganizationsList) Fetch(token string) error {
	err := list.Page.Fetch(token)
	if err != nil {
		return err
	}

	updatedList := newOrganizationsListFromPage(list.config, list.plan, list.Page)
	list.TotalResults = updatedList.TotalResults
	list.TotalPages = updatedList.TotalPages
	list.NextURL = updatedList.NextURL
	list.PrevURL = updatedList.PrevURL
	list.Organizations = updatedList.Organizations

	return nil
}

func newOrganizationsListFromPage(config Config, plan requestPlan, page Page) OrganizationsList {
	list := NewOrganizationsList(config, plan)
	list.TotalResults = page.TotalResults
	list.TotalPages = page.TotalPages
	list.PrevURL = page.PrevURL
	list.NextURL = page.NextURL
	list.Organizations = make([]Organization, 0)

	for _, orgResource := range page.Resources {
		var orgResponse documents.OrganizationResponse
		err := json.Unmarshal(orgResource, &orgResponse)
		if err != nil {
			panic(err)
		}

		list.Organizations = append(list.Organizations, newOrganizationFromResponse(config, orgResponse))
	}

	return list
}
