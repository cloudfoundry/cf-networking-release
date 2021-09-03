package rainmaker

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/pivotal-cf-experimental/rainmaker/internal/documents"
	"github.com/pivotal-cf-experimental/rainmaker/internal/network"
)

type Page struct {
	config       Config
	plan         requestPlan
	TotalResults int
	TotalPages   int
	NextURL      string
	PrevURL      string
	Resources    []json.RawMessage
}

func NewPage(config Config, plan requestPlan) Page {
	return Page{
		config: config,
		plan:   plan,
	}
}

func newPageFromResponse(config Config, plan requestPlan, resp documents.PageResponse) Page {
	return Page{
		config:       config,
		plan:         plan,
		TotalResults: resp.TotalResults,
		TotalPages:   resp.TotalPages,
		NextURL:      resp.NextURL,
		PrevURL:      resp.PrevURL,
		Resources:    resp.Resources,
	}
}

func (p Page) Next(token string) (Page, error) {
	nextURL, err := url.Parse("http://example.com" + p.NextURL)
	if err != nil {
		return Page{}, err
	}

	return NewPage(p.config, newRequestPlan(nextURL.Path, nextURL.Query())), nil
}

func (p Page) Prev(token string) (Page, error) {
	prevURL, err := url.Parse("http://example.com" + p.PrevURL)
	if err != nil {
		return Page{}, err
	}

	return NewPage(p.config, newRequestPlan(prevURL.Path, prevURL.Query())), nil
}

func (p Page) HasNextPage() bool {
	return p.NextURL != ""
}

func (p Page) HasPrevPage() bool {
	return p.PrevURL != ""
}

func (p *Page) Fetch(token string) error {
	u := url.URL{
		Path:     p.plan.Path,
		RawQuery: p.plan.Query.Encode(),
	}

	resp, err := newNetworkClient(p.config).MakeRequest(network.Request{
		Method:                "GET",
		Path:                  u.String(),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return err
	}

	var response documents.PageResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		panic(err)
	}

	updatedPage := newPageFromResponse(p.config, p.plan, response)
	p.TotalResults = updatedPage.TotalResults
	p.TotalPages = updatedPage.TotalPages
	p.NextURL = updatedPage.NextURL
	p.PrevURL = updatedPage.PrevURL
	p.Resources = updatedPage.Resources

	return nil
}
