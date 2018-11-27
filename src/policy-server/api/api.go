package api

import "policy-server/store"

var ICMPDefault = -1

//go:generate counterfeiter -o fakes/policy_mapper.go --fake-name PolicyMapper . PolicyMapper
type PolicyMapper interface {
	AsStorePolicy([]byte) ([]store.Policy, error) // marshal
	AsBytes([]store.Policy) ([]byte, error)       // unmarshal
}

//go:generate counterfeiter -o fakes/policy_collection_writer.go --fake-name PolicyCollectionWriter . PolicyCollectionWriter
type PolicyCollectionWriter interface {
	AsBytes([]store.Policy, []store.EgressPolicy) ([]byte, error) // unmarshal
}

type PolicyCollectionPayload struct {
	TotalPolicies       int            `json:"total_policies"`
	Policies            []Policy       `json:"policies"`
	TotalEgressPolicies int            `json:"total_egress_policies,omitempty"`
	EgressPolicies      []EgressPolicy `json:"egress_policies,omitempty"`
}

type PoliciesPayload struct {
	TotalPolicies int      `json:"total_policies"`
	Policies      []Policy `json:"policies"`
}

type EgressPoliciesPayload struct {
	TotalEgressPolicies int            `json:"total_egress_policies,omitempty"`
	EgressPolicies      []EgressPolicy `json:"egress_policies,omitempty"`
}

type Policy struct {
	Source      Source      `json:"source"`
	Destination Destination `json:"destination"`
}

type EgressPolicy struct {
	ID           string             `json:"id,omitempty"`
	Source       *EgressSource      `json:"source"`
	Destination  *EgressDestination `json:"destination"`
	AppLifecycle string             `json:"app_lifecycle"`
}

type EgressSource struct {
	ID   string `json:"id"`
	Type string `json:"type,omitempty"`
}

type EgressDestination struct {
	GUID        string    `json:"id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Protocol    string    `json:"protocol,omitempty"`
	Ports       []Ports   `json:"ports,omitempty"`
	IPRanges    []IPRange `json:"ips,omitempty"`
	ICMPType    *int      `json:"icmp_type,omitempty"`
	ICMPCode    *int      `json:"icmp_code,omitempty"`
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
