package store

type PolicyCollection struct {
	Policies       []Policy
	EgressPolicies []EgressPolicy
}

type Policy struct {
	Source      Source
	Destination Destination
}

type Source struct {
	ID  string
	Tag string
}

type Destination struct {
	ID       string
	Tag      string
	Protocol string
	Port     int
	Ports    Ports
}

type Ports struct {
	Start int
	End   int
}

type Tag struct {
	ID   string
	Tag  string
	Type string
}

type EgressPolicy struct {
	Source      EgressSource
	Destination EgressDestination
}

type EgressSource struct {
	ID string
}

type EgressDestination struct {
	Protocol string
	IPRanges []IPRange
}

type IPRange struct {
	Start string
	End   string
}
