package store

type Store struct {
	LocalAddress       string
	addresses          map[string]int
	stalenessThreshold int
}

func New(localAddress string, stalenessThreshold int) *Store {
	return &Store{
		LocalAddress:       localAddress,
		addresses:          make(map[string]int),
		stalenessThreshold: stalenessThreshold,
	}
}

func (s *Store) Add(addrs []string) {
	for addr, count := range s.addresses {
		s.addresses[addr] = count + 1
	}

	for _, addr := range addrs {
		s.addresses[addr] = 0
	}
}

func (s *Store) GetAddresses() []string {
	s.addresses[s.LocalAddress] = 0

	var addrs []string
	for addr, count := range s.addresses {
		if count < s.stalenessThreshold {
			addrs = append(addrs, addr)
		} else {
			delete(s.addresses, addr)
		}
	}
	return addrs
}
