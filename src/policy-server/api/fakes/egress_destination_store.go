// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"policy-server/api"
	"policy-server/store"
	"sync"
)

type EgressDestinationStore struct {
	GetByGUIDStub        func(guid ...string) ([]store.EgressDestination, error)
	getByGUIDMutex       sync.RWMutex
	getByGUIDArgsForCall []struct {
		guid []string
	}
	getByGUIDReturns struct {
		result1 []store.EgressDestination
		result2 error
	}
	getByGUIDReturnsOnCall map[int]struct {
		result1 []store.EgressDestination
		result2 error
	}
	GetByNameStub        func(name ...string) ([]store.EgressDestination, error)
	getByNameMutex       sync.RWMutex
	getByNameArgsForCall []struct {
		name []string
	}
	getByNameReturns struct {
		result1 []store.EgressDestination
		result2 error
	}
	getByNameReturnsOnCall map[int]struct {
		result1 []store.EgressDestination
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *EgressDestinationStore) GetByGUID(guid ...string) ([]store.EgressDestination, error) {
	fake.getByGUIDMutex.Lock()
	ret, specificReturn := fake.getByGUIDReturnsOnCall[len(fake.getByGUIDArgsForCall)]
	fake.getByGUIDArgsForCall = append(fake.getByGUIDArgsForCall, struct {
		guid []string
	}{guid})
	fake.recordInvocation("GetByGUID", []interface{}{guid})
	fake.getByGUIDMutex.Unlock()
	if fake.GetByGUIDStub != nil {
		return fake.GetByGUIDStub(guid...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.getByGUIDReturns.result1, fake.getByGUIDReturns.result2
}

func (fake *EgressDestinationStore) GetByGUIDCallCount() int {
	fake.getByGUIDMutex.RLock()
	defer fake.getByGUIDMutex.RUnlock()
	return len(fake.getByGUIDArgsForCall)
}

func (fake *EgressDestinationStore) GetByGUIDArgsForCall(i int) []string {
	fake.getByGUIDMutex.RLock()
	defer fake.getByGUIDMutex.RUnlock()
	return fake.getByGUIDArgsForCall[i].guid
}

func (fake *EgressDestinationStore) GetByGUIDReturns(result1 []store.EgressDestination, result2 error) {
	fake.GetByGUIDStub = nil
	fake.getByGUIDReturns = struct {
		result1 []store.EgressDestination
		result2 error
	}{result1, result2}
}

func (fake *EgressDestinationStore) GetByGUIDReturnsOnCall(i int, result1 []store.EgressDestination, result2 error) {
	fake.GetByGUIDStub = nil
	if fake.getByGUIDReturnsOnCall == nil {
		fake.getByGUIDReturnsOnCall = make(map[int]struct {
			result1 []store.EgressDestination
			result2 error
		})
	}
	fake.getByGUIDReturnsOnCall[i] = struct {
		result1 []store.EgressDestination
		result2 error
	}{result1, result2}
}

func (fake *EgressDestinationStore) GetByName(name ...string) ([]store.EgressDestination, error) {
	fake.getByNameMutex.Lock()
	ret, specificReturn := fake.getByNameReturnsOnCall[len(fake.getByNameArgsForCall)]
	fake.getByNameArgsForCall = append(fake.getByNameArgsForCall, struct {
		name []string
	}{name})
	fake.recordInvocation("GetByName", []interface{}{name})
	fake.getByNameMutex.Unlock()
	if fake.GetByNameStub != nil {
		return fake.GetByNameStub(name...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.getByNameReturns.result1, fake.getByNameReturns.result2
}

func (fake *EgressDestinationStore) GetByNameCallCount() int {
	fake.getByNameMutex.RLock()
	defer fake.getByNameMutex.RUnlock()
	return len(fake.getByNameArgsForCall)
}

func (fake *EgressDestinationStore) GetByNameArgsForCall(i int) []string {
	fake.getByNameMutex.RLock()
	defer fake.getByNameMutex.RUnlock()
	return fake.getByNameArgsForCall[i].name
}

func (fake *EgressDestinationStore) GetByNameReturns(result1 []store.EgressDestination, result2 error) {
	fake.GetByNameStub = nil
	fake.getByNameReturns = struct {
		result1 []store.EgressDestination
		result2 error
	}{result1, result2}
}

func (fake *EgressDestinationStore) GetByNameReturnsOnCall(i int, result1 []store.EgressDestination, result2 error) {
	fake.GetByNameStub = nil
	if fake.getByNameReturnsOnCall == nil {
		fake.getByNameReturnsOnCall = make(map[int]struct {
			result1 []store.EgressDestination
			result2 error
		})
	}
	fake.getByNameReturnsOnCall[i] = struct {
		result1 []store.EgressDestination
		result2 error
	}{result1, result2}
}

func (fake *EgressDestinationStore) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.getByGUIDMutex.RLock()
	defer fake.getByGUIDMutex.RUnlock()
	fake.getByNameMutex.RLock()
	defer fake.getByNameMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *EgressDestinationStore) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ api.EgressDestinationStore = new(EgressDestinationStore)