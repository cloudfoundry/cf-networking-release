package models

type Policy struct {
	Source      Source      `json:"source"`
	Destination Destination `json:"destination"`
}

type Source struct {
	ID string `json:"id"`
}

type Destination struct {
	ID       string `json:"id"`
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
}

type Tag struct {
	ID  string `json:"id"`
	Tag string `json:"tag"`
}
