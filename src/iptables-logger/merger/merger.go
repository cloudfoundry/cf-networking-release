package merger

import (
	"fmt"
	"iptables-logger/parser"
	"iptables-logger/repository"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o fakes/containerRepo.go --fake-name ContainerRepo . containerRepo
type containerRepo interface {
	GetByIP(string) (repository.Container, error)
}

type IPTablesLogData struct {
	Message string
	Data    lager.Data
}

type Merger struct {
	ContainerRepo containerRepo
}

func (m *Merger) Merge(parsedData parser.ParsedData) (IPTablesLogData, error) {
	message := parsedData.Direction
	if parsedData.Allowed {
		message += "-allowed"
	} else {
		message += "-denied"
	}

	var key, ipToLookup string
	if parsedData.Direction == "ingress" {
		key = "destination"
		ipToLookup = parsedData.DestinationIP
	} else {
		key = "source"
		ipToLookup = parsedData.SourceIP
	}

	containerData, err := m.ContainerRepo.GetByIP(ipToLookup)
	if err != nil {
		return IPTablesLogData{}, fmt.Errorf("get container by ip: %s", err)
	}
	return IPTablesLogData{
		Message: message,
		Data: lager.Data{
			key:      containerData,
			"packet": parsedData,
		},
	}, nil
}
