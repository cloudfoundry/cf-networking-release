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

type CNIAddResult struct {
	ContainerID string `json:"container_id"`
	GroupID     string `json:"group_id"`
	IP          string `json:"ip"`
}

type CNIDelResult struct {
	ContainerID string `json:"container_id"`
}
