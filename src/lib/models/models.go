package models

type Container struct {
	ID string
	IP string
}

type CNIAddResult struct {
	ContainerID string `json:"container_id"`
	GroupID     string `json:"group_id"`
	IP          string `json:"ip"`
}

type CNIDelResult struct {
	ContainerID string `json:"container_id"`
}
