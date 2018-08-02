package api

import "policy-server/store"

//go:generate counterfeiter -o fakes/policy_mapper.go --fake-name PolicyMapper . PolicyMapper
type PolicyMapper interface {
	AsStorePolicy([]byte) (store.PolicyCollection, error)         // marshal
	AsBytes([]store.Policy, []store.EgressPolicy) ([]byte, error) // unmarshal
}

type PoliciesPayload struct {
	TotalPolicies       int            `json:"total_policies"`
	Policies            []Policy       `json:"policies"`
	TotalEgressPolicies int            `json:"total_egress_policies,omitempty"`
	EgressPolicies      []EgressPolicy `json:"egress_policies,omitempty"`
}

type Policy struct {
	Source      Source      `json:"source"`
	Destination Destination `json:"destination"`
}

type EgressPolicy struct {
	Source      *EgressSource      `json:"source"`
	Destination *EgressDestination `json:"destination"`
}

type EgressSource struct {
	ID string `json:"id"`
}

type EgressDestination struct {
	Protocol string    `json:"protocol"`
	Ports    []Ports   `json:"ports,omitempty"`
	IPRanges []IPRange `json:"ips"`
	ICMPType *int      `json:"type,omitempty"`
	ICMPCode *int      `json:"code,omitempty"`
}

type Source struct {
	ID   string `json:"id"`
	Tag  string `json:"tag,omitempty"`
	Type string `json:"type,omitempty"`
}

type Destination struct {
	ID       string    `json:"id"`
	Tag      string    `json:"tag,omitempty"`
	Protocol string    `json:"protocol"`
	Ports    Ports     `json:"ports"`
	Type     string    `json:"type,omitempty"`
	IPs      []IPRange `json:"ips,omitempty"`
}

type IPRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type Ports struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type Tag struct {
	ID   string `json:"id"`
	Tag  string `json:"tag"`
	Type string `json:"type"`
}

type Space struct {
	Name    string `json:"name"`
	OrgGUID string `json:"organization_guid"`
}
