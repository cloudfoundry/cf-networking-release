package rainmaker

import "net/url"

type requestPlan struct {
	Path  string
	Query url.Values
}

func newRequestPlan(path string, query url.Values) requestPlan {
	return requestPlan{
		Path:  path,
		Query: query,
	}
}
