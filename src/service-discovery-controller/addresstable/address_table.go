package addresstable

import (
	"fmt"
	"sync"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type AddressTable struct {
	addresses          map[string][]entry
	clock              clock.Clock
	stalenessThreshold time.Duration
	mutex              sync.RWMutex
	ticker             clock.Ticker
	pausedPruning      bool
	logger             lager.Logger
	lastResume         time.Time
	resumePruningDelay time.Duration
	warm               bool
	warmMutex          sync.RWMutex
}

type entry struct {
	ip         string
	updateTime time.Time
}

func NewAddressTable(stalenessThreshold, pruningInterval, resumePruningDelay time.Duration, clock clock.Clock, logger lager.Logger) *AddressTable {
	table := &AddressTable{
		addresses:          map[string][]entry{},
		clock:              clock,
		stalenessThreshold: stalenessThreshold,
		ticker:             clock.NewTicker(pruningInterval),
		pausedPruning:      false,
		logger:             logger,
		resumePruningDelay: resumePruningDelay,
	}

	table.pruneStaleEntriesOnInterval(pruningInterval)
	return table
}

func (at *AddressTable) Add(hostnames []string, ip string) {
	at.mutex.Lock()
	for _, hostname := range hostnames {
		fqHostname := fqdn(hostname)
		entries := at.entriesForHostname(fqHostname)
		entryIndex := indexOf(entries, ip)
		if entryIndex == -1 {
			at.addresses[fqHostname] = append(entries, entry{ip: ip, updateTime: at.clock.Now()})
		} else {
			at.addresses[fqHostname][entryIndex].updateTime = at.clock.Now()
		}
	}
	at.mutex.Unlock()
}

func (at *AddressTable) Remove(hostnames []string, ip string) {
	at.mutex.Lock()
	for _, hostname := range hostnames {
		fqHostname := fqdn(hostname)
		entries := at.entriesForHostname(fqHostname)
		index := indexOf(entries, ip)
		if index > -1 {
			if len(entries) == 1 {
				delete(at.addresses, fqHostname)
			} else {
				at.addresses[fqHostname] = append(entries[:index], entries[index+1:]...)
			}
		}
	}
	at.mutex.Unlock()
}

func (at *AddressTable) Lookup(hostname string) []string {
	at.mutex.RLock()

	found := at.entriesForHostname(fqdn(hostname))
	ips := entriesToIPs(found)

	at.mutex.RUnlock()

	return ips
}

func (at *AddressTable) GetAllAddresses() map[string][]string {
	at.mutex.RLock()

	addresses := map[string][]string{}
	for address, entries := range at.addresses {
		addresses[address] = entriesToIPs(entries)
	}

	at.mutex.RUnlock()

	return addresses
}

func (at *AddressTable) SetWarm() {
	at.warmMutex.Lock()
	at.warm = true
	at.warmMutex.Unlock()
}

func (at *AddressTable) IsWarm() bool {
	at.warmMutex.RLock()
	warm := at.warm
	at.warmMutex.RUnlock()

	return warm
}

func (at *AddressTable) Shutdown() {
	at.ticker.Stop()
}

func (at *AddressTable) PausePruning() {
	at.logger.Info("pruning-pause")
	at.mutex.Lock()
	at.pausedPruning = true
	at.mutex.Unlock()
}

func (at *AddressTable) ResumePruning() {
	at.logger.Info("pruning-resume")
	at.mutex.Lock()
	at.lastResume = at.clock.Now()
	at.pausedPruning = false
	at.mutex.Unlock()
}

func (at *AddressTable) entriesForHostname(hostname string) []entry {
	if existing, ok := at.addresses[hostname]; ok {
		return existing
	} else {
		return []entry{}
	}
}

func entriesToIPs(entries []entry) []string {
	ips := make([]string, len(entries))
	for idx, entry := range entries {
		ips[idx] = entry.ip
	}

	return ips
}

func (at *AddressTable) pruneStaleEntriesOnInterval(pruningInterval time.Duration) {
	go func() {
		defer at.ticker.Stop()
		for _ = range at.ticker.C() {
			at.mutex.RLock()
			if at.pausedPruning || (at.clock.Since(at.lastResume) < at.resumePruningDelay) {
				at.mutex.RUnlock()
				continue
			}
			at.mutex.RUnlock()
			staleAddresses := at.addressesWithStaleEntriesWithReadLock()
			at.pruneStaleEntriesWithWriteLock(staleAddresses)
		}
	}()
}

func (at *AddressTable) pruneStaleEntriesWithWriteLock(candidateAddresses []string) {
	if len(candidateAddresses) == 0 {
		return
	}

	var oldTotal, newTotal int
	at.mutex.Lock()
	for _, staleAddr := range candidateAddresses {
		entries, ok := at.addresses[staleAddr]
		if ok {
			oldCount := len(entries)
			freshEntries := []entry{}
			for _, entry := range entries {
				if at.clock.Since(entry.updateTime) <= at.stalenessThreshold {
					freshEntries = append(freshEntries, entry)
				} else {
					at.logger.Debug(fmt.Sprintf("pruning address %s from %s", entry.ip, staleAddr))
				}
			}
			at.addresses[staleAddr] = freshEntries
			newCount := len(freshEntries)
			oldTotal += oldCount
			newTotal += newCount
		}
	}
	at.mutex.Unlock()
	at.logger.Info("pruned", lager.Data{"old-total": oldTotal, "new-total": newTotal})
}

func (at *AddressTable) addressesWithStaleEntriesWithReadLock() []string {
	staleAddresses := []string{}
	at.mutex.RLock()
	for address, entries := range at.addresses {
		for _, entry := range entries {
			if at.clock.Since(entry.updateTime) > at.stalenessThreshold {
				staleAddresses = append(staleAddresses, address)
				break
			}
		}
	}
	at.mutex.RUnlock()
	return staleAddresses
}

func indexOf(entries []entry, value string) int {
	for idx, entry := range entries {
		if entry.ip == value {
			return idx
		}
	}
	return -1
}

func isFqdn(s string) bool {
	l := len(s)
	if l == 0 {
		return false
	}
	return s[l-1] == '.'
}

func fqdn(s string) string {
	if isFqdn(s) {
		return s
	}
	return s + "."
}
