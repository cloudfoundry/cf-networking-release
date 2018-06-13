package api

import "policy-server/store"

//go:generate counterfeiter -o fakes/policy_mapper.go --fake-name PolicyMapper . PolicyMapper
type PolicyMapper interface {
	AsStorePolicy([]byte) ([]store.Policy, error) // marshal
	AsBytes([]store.Policy) ([]byte, error)       // unmarshal
}

type Policies struct {
	TotalPolicies int      `json:"total_policies"`
	Policies      []Policy `json:"policies"`
}

type Policy struct {
	Source      Source      `json:"source"`
	Destination Destination `json:"destination"`
}

type Source struct {
	ID  string `json:"id"`
	Tag string `json:"tag,omitempty"`
}

type Destination struct {
	ID       string `json:"id"`
	Tag      string `json:"tag,omitempty"`
	Protocol string `json:"protocol"`
	Ports    Ports  `json:"ports"`
}

type Ports struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type Tag struct {
	ID  string `json:"id"`
	Tag string `json:"tag"`
	Type string `json:"type"`
}

type Space struct {
	Name    string `json:"name"`
	OrgGUID string `json:"organization_guid"`
}
