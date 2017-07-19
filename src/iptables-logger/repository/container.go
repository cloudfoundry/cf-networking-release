package repository

import (
	"fmt"
	"lib/datastore"
)

type Container struct {
	Handle   string `json:"container_id"`
	AppID    string `json:"app_guid"`
	SpaceID  string `json:"space_guid"`
	OrgID    string `json:"organization_guid"`
	HostIp   string `json:"host_ip"`
	HostGuid string `json:"host_guid"`
}

type ContainerRepo struct {
	Store datastore.Datastore
}

func (c *ContainerRepo) GetByIP(ip string) (Container, error) {
	containers, err := c.Store.ReadAll()
	if err != nil {
		return Container{}, fmt.Errorf("read all: %s", err)
	}

	for _, container := range containers {
		if container.IP == ip {
			appID, ok := container.Metadata["app_id"].(string)
			if !ok {
				appID = ""
			}
			spaceID, ok := container.Metadata["space_id"].(string)
			if !ok {
				spaceID = ""
			}
			orgID, ok := container.Metadata["org_id"].(string)
			if !ok {
				orgID = ""
			}
			return Container{
				Handle:  container.Handle,
				AppID:   appID,
				SpaceID: spaceID,
				OrgID:   orgID,
			}, nil
		}
	}

	return Container{}, nil
}
