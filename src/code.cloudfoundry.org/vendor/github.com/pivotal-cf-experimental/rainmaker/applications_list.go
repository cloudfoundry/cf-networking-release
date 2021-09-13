package rainmaker

import (
	"encoding/json"

	"github.com/pivotal-cf-experimental/rainmaker/internal/documents"
)

type ApplicationsList struct {
	config Config
	plan   requestPlan
	page   Page

	TotalResults int
	TotalPages   int
	NextURL      string
	PrevURL      string
	Applications []Application
}

func NewApplicationsList(config Config, plan requestPlan) ApplicationsList {
	return ApplicationsList{
		config: config,
		plan:   plan,
		page:   NewPage(config, plan),
	}
}

func (list *ApplicationsList) Fetch(token string) error {
	err := list.page.Fetch(token)
	if err != nil {
		return err
	}

	updatedList, err := newApplicationsListFromPage(list.config, list.plan, list.page)
	if err != nil {
		return err
	}

	list.TotalResults = updatedList.TotalResults
	list.TotalPages = updatedList.TotalPages
	list.NextURL = updatedList.NextURL
	list.PrevURL = updatedList.PrevURL
	list.Applications = updatedList.Applications

	return nil
}

func (list ApplicationsList) HasNextPage() bool {
	return list.NextURL != ""
}

func (list ApplicationsList) HasPrevPage() bool {
	return list.PrevURL != ""
}

func (list ApplicationsList) Next(token string) (ApplicationsList, error) {
	nextPage, err := list.page.Next(token)
	if err != nil {
		return ApplicationsList{}, err
	}

	nextList, err := newApplicationsListFromPage(list.config, nextPage.plan, nextPage)
	if err != nil {
		return ApplicationsList{}, err
	}

	err = nextList.Fetch(token)

	return nextList, err
}

func (list ApplicationsList) Prev(token string) (ApplicationsList, error) {
	prevPage, err := list.page.Prev(token)
	if err != nil {
		return ApplicationsList{}, err
	}

	prevList, err := newApplicationsListFromPage(list.config, prevPage.plan, prevPage)
	if err != nil {
		return ApplicationsList{}, err
	}

	err = prevList.Fetch(token)

	return prevList, err
}

func newApplicationsListFromPage(config Config, plan requestPlan, page Page) (ApplicationsList, error) {
	list := NewApplicationsList(config, plan)
	list.TotalResults = page.TotalResults
	list.TotalPages = page.TotalPages
	list.PrevURL = page.PrevURL
	list.NextURL = page.NextURL
	list.Applications = make([]Application, 0)

	for _, appResource := range page.Resources {
		var appResponse documents.ApplicationResponse
		err := json.Unmarshal(appResource, &appResponse)
		if err != nil {
			return ApplicationsList{}, err
		}

		list.Applications = append(list.Applications, newApplicationFromResponse(config, appResponse))
	}

	return list, nil
}
