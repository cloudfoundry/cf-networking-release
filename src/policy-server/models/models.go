package models

import (
	"encoding/json"
	"errors"
)

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
	Tag      string `json:"tag,omitempty"`
	Protocol string `json:"protocol"`
	Port     int    `json:"port,omitempty"`
	Ports    Ports  `json:"ports"`
}

type Ports struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type Tag struct {
	ID  string `json:"id"`
	Tag string `json:"tag"`
}

type Space struct {
	Name    string `json:name`
	OrgGUID string `json:organization_guid`
}

type destination struct {
	ID       string `json:"id"`
	Tag      string `json:"tag,omitempty"`
	Protocol string `json:"protocol"`
	Port     int    `json:"port,omitempty"`
	Ports    Ports  `json:"ports"`
}

func fixPorts(d *Destination) error {
	hasPort := d.Port > 0
	hasPorts := d.Ports.Start > 0 || d.Ports.End > 0
	if hasPort && !hasPorts {
		d.Ports.Start = d.Port
		d.Ports.End = d.Port
	} else if !hasPort && hasPorts && d.Ports.Start == d.Ports.End {
		d.Port = d.Ports.Start
	} else if hasPort && !(d.Ports.Start == d.Ports.End && d.Port == d.Ports.Start) {
		return errors.New("ports and port mismatch")
	}
	return nil
}

func (d Destination) MarshalJSON() ([]byte, error) {
	err := fixPorts(&d)
	if err != nil {
		return []byte{}, err
	}
	dest := destination{
		ID:       d.ID,
		Tag:      d.Tag,
		Protocol: d.Protocol,
		Port:     d.Port,
		Ports:    d.Ports,
	}

	return json.Marshal(dest) // error not tested
}

func (d *Destination) UnmarshalJSON(input []byte) error {
	dest := destination{}

	err := json.Unmarshal(input, &dest)
	if err != nil {
		return err // not tested
	}

	d.ID = dest.ID
	d.Tag = dest.Tag
	d.Protocol = dest.Protocol
	d.Port = dest.Port
	d.Ports = dest.Ports

	err = fixPorts(d)
	if err != nil {
		return err
	}

	return nil
}
