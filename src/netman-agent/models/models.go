package models

type Containers map[string][]Container

type Container struct {
	ID string
	IP string
}

type Policy struct {
	Source      Source
	Destination Destination
}

type Source struct {
	ID string
}

type Destination struct {
	ID       string
	Port     int
	Protocol string
}
