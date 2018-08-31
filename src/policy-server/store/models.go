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
	ID   string
	Type string
}

type EgressDestination struct {
	ID          string
	Name        string
	Description string
	Protocol    string
	Ports       []Ports
	IPRanges    []IPRange
	ICMPType    int
	ICMPCode    int
}

type IPRange struct {
	Start string
	End   string
}

type EgressPolicyIDCollection struct {
	EgressPolicyID        int64
	DestinationTerminalID int64
	DestinationIPRangeID  int64
	SourceTerminalID      int64
	SourceAppID           int64
	SourceSpaceID         int64
}
