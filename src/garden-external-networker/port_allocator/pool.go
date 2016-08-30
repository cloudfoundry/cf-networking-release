package port_allocator

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"code.cloudfoundry.org/lager"
)

var ErrorPortPoolExhausted = errors.New("port pool exhausted")

type Pool struct {
	AcquiredPorts map[int]bool
}

func (p *Pool) MarshalJSON() ([]byte, error) {
	portList := []int{}
	for port, _ := range p.AcquiredPorts {
		portList = append(portList, port)
	}
	var portPool struct {
		AcquiredPorts []int `json:"acquired_ports"`
	}
	sort.Ints(portList)
	portPool.AcquiredPorts = portList
	return json.Marshal(portPool)

}

func (p *Pool) UnmarshalJSON(bytes []byte) error {
	var portPool struct {
		AcquiredPorts []int `json:"acquired_ports"`
	}
	err := json.Unmarshal(bytes, &portPool)
	if err != nil {
		return err
	}
	p.AcquiredPorts = make(map[int]bool)
	for _, port := range portPool.AcquiredPorts {
		p.AcquiredPorts[port] = true
	}
	return nil
}

type Tracker struct {
	Logger    lager.Logger
	StartPort int
	Capacity  int
}

func (t *Tracker) InRange(port int) bool {
	return port >= t.StartPort && port < t.StartPort+t.Capacity
}

func (t *Tracker) AcquireOne(pool *Pool) (int, error) {
	if pool.AcquiredPorts == nil {
		pool.AcquiredPorts = make(map[int]bool)
	}

	for i := 0; i < t.Capacity; i++ {
		candidatePort := t.StartPort + i
		if !contains(pool.AcquiredPorts, candidatePort) {
			pool.AcquiredPorts[candidatePort] = true
			return candidatePort, nil
		}
	}
	return -1, ErrorPortPoolExhausted
}

func (t *Tracker) ReleaseMany(pool *Pool, toRelease []int) error {
	for _, port := range toRelease {
		if port < t.StartPort || port >= t.StartPort+t.Capacity {
			t.Logger.Error("release-many", fmt.Errorf("releasing port out of range"),
				lager.Data{"start-port": t.StartPort, "capacity": t.Capacity, "to-release": port})
			continue
		}
		_, ok := pool.AcquiredPorts[port]
		if !ok {
			t.Logger.Error("release-many", fmt.Errorf("port %d was not previously acquired", port),
				lager.Data{"to-release": port})
			continue
		}
		delete(pool.AcquiredPorts, port)
	}
	return nil
}

func contains(list map[int]bool, candidate int) bool {
	_, ok := list[candidate]
	return ok
}
