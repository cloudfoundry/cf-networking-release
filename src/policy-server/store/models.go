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
	ID           string
	Source       EgressSource
	Destination  EgressDestination
	AppLifecycle string
}

type EgressSource struct {
	TerminalGUID string
	ID           string
	Type         string
}

type EgressDestination struct {
	GUID        string
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
