package store

import "sync"

//go:generate counterfeiter -o ../fakes/mutex.go --fake-name Mutex . mutex
type mutex interface {
	sync.Locker
}

type Store struct {
	LocalAddress       string
	addresses          map[string]int
	stalenessThreshold int
	locker             sync.Locker
}

func New(localAddress string, stalenessThreshold int, locker sync.Locker) *Store {
	return &Store{
		LocalAddress:       localAddress,
		addresses:          make(map[string]int),
		stalenessThreshold: stalenessThreshold,
		locker:             locker,
	}
}

func (s *Store) Add(addrs []string) {
	s.locker.Lock()
	defer s.locker.Unlock()

	for addr, staleness := range s.addresses {
		s.addresses[addr] = staleness + 1
	}

	for _, addr := range addrs {
		s.addresses[addr] = 0
	}
}

func (s *Store) GetAddresses() []string {
	s.locker.Lock()
	defer s.locker.Unlock()

	s.addresses[s.LocalAddress] = 0

	var addrs []string
	for addr, staleness := range s.addresses {
		if staleness < s.stalenessThreshold {
			addrs = append(addrs, addr)
		} else {
			delete(s.addresses, addr)
		}
	}
	return addrs
}
