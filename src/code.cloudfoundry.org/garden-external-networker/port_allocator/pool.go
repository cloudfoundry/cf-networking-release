package port_allocator

import (
	"encoding/json"
	"errors"
)

var ErrorPortPoolExhausted = errors.New("port pool exhausted")

type Pool struct {
	AcquiredPorts map[uint32]string
}

func (p *Pool) MarshalJSON() ([]byte, error) {
	var jsonData struct {
		AcquiredPorts map[string][]uint32 `json:"acquired_ports"`
	}
	jsonData.AcquiredPorts = make(map[string][]uint32)

	for port, handle := range p.AcquiredPorts {
		jsonData.AcquiredPorts[handle] = append(jsonData.AcquiredPorts[handle], port)
	}
	return json.Marshal(jsonData)
}

func (p *Pool) UnmarshalJSON(bytes []byte) error {
	var jsonData struct {
		AcquiredPorts map[string][]uint32 `json:"acquired_ports"`
	}
	err := json.Unmarshal(bytes, &jsonData)
	if err != nil {
		return err
	}

	p.AcquiredPorts = make(map[uint32]string)
	for handle, ports := range jsonData.AcquiredPorts {
		for _, port := range ports {
			p.AcquiredPorts[port] = handle
		}
	}
	return nil
}

type Tracker struct {
	StartPort uint32
	Capacity  uint32
}

func (t *Tracker) InRange(port uint32) bool {
	return port >= t.StartPort && port < t.StartPort+t.Capacity
}

func (t *Tracker) AcquireOne(pool *Pool, handler string) (uint32, error) {
	if pool.AcquiredPorts == nil {
		pool.AcquiredPorts = make(map[uint32]string)
	}

	for i := uint32(0); i < t.Capacity; i++ {
		candidatePort := t.StartPort + i
		if !contains(pool.AcquiredPorts, candidatePort) {
			pool.AcquiredPorts[candidatePort] = handler
			return candidatePort, nil
		}
	}
	return 0, ErrorPortPoolExhausted
}

func (t *Tracker) ReleaseAll(pool *Pool, handle string) error {
	for port, h := range pool.AcquiredPorts {
		if h == handle {
			delete(pool.AcquiredPorts, port)
		}
	}
	return nil
}

func contains(list map[uint32]string, candidate uint32) bool {
	_, ok := list[candidate]
	return ok
}
