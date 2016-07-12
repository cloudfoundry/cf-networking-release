package models

type Container struct {
	ID string
	IP string
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
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type CNIAddResult struct {
	ContainerID string `json:"container_id"`
	GroupID     string `json:"group_id"`
	IP          string `json:"ip"`
}

type CNIDelResult struct {
	ContainerID string `json:"container_id"`
}
