// This file was generated by counterfeiter
package fakes

import (
	"context"
	"policy-server/models"
	"policy-server/store"
	"sync"
)

type Store struct {
	CreateStub        func(context.Context, []models.Policy) error
	createMutex       sync.RWMutex
	createArgsForCall []struct {
		arg1 context.Context
		arg2 []models.Policy
	}
	createReturns struct {
		result1 error
	}
	createReturnsOnCall map[int]struct {
		result1 error
	}
	AllStub        func() ([]models.Policy, error)
	allMutex       sync.RWMutex
	allArgsForCall []struct{}
	allReturns     struct {
		result1 []models.Policy
		result2 error
	}
	allReturnsOnCall map[int]struct {
		result1 []models.Policy
		result2 error
	}
	DeleteStub        func([]models.Policy) error
	deleteMutex       sync.RWMutex
	deleteArgsForCall []struct {
		arg1 []models.Policy
	}
	deleteReturns struct {
		result1 error
	}
	deleteReturnsOnCall map[int]struct {
		result1 error
	}
	TagsStub        func() ([]models.Tag, error)
	tagsMutex       sync.RWMutex
	tagsArgsForCall []struct{}
	tagsReturns     struct {
		result1 []models.Tag
		result2 error
	}
	tagsReturnsOnCall map[int]struct {
		result1 []models.Tag
		result2 error
	}
	ByGuidsStub        func([]string, []string) ([]models.Policy, error)
	byGuidsMutex       sync.RWMutex
	byGuidsArgsForCall []struct {
		arg1 []string
		arg2 []string
	}
	byGuidsReturns struct {
		result1 []models.Policy
		result2 error
	}
	byGuidsReturnsOnCall map[int]struct {
		result1 []models.Policy
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *Store) Create(arg1 context.Context, arg2 []models.Policy) error {
	var arg2Copy []models.Policy
	if arg2 != nil {
		arg2Copy = make([]models.Policy, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.createMutex.Lock()
	ret, specificReturn := fake.createReturnsOnCall[len(fake.createArgsForCall)]
	fake.createArgsForCall = append(fake.createArgsForCall, struct {
		arg1 context.Context
		arg2 []models.Policy
	}{arg1, arg2Copy})
	fake.recordInvocation("Create", []interface{}{arg1, arg2Copy})
	fake.createMutex.Unlock()
	if fake.CreateStub != nil {
		return fake.CreateStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.createReturns.result1
}

func (fake *Store) CreateCallCount() int {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	return len(fake.createArgsForCall)
}

func (fake *Store) CreateArgsForCall(i int) (context.Context, []models.Policy) {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	return fake.createArgsForCall[i].arg1, fake.createArgsForCall[i].arg2
}

func (fake *Store) CreateReturns(result1 error) {
	fake.CreateStub = nil
	fake.createReturns = struct {
		result1 error
	}{result1}
}

func (fake *Store) CreateReturnsOnCall(i int, result1 error) {
	fake.CreateStub = nil
	if fake.createReturnsOnCall == nil {
		fake.createReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.createReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *Store) All() ([]models.Policy, error) {
	fake.allMutex.Lock()
	ret, specificReturn := fake.allReturnsOnCall[len(fake.allArgsForCall)]
	fake.allArgsForCall = append(fake.allArgsForCall, struct{}{})
	fake.recordInvocation("All", []interface{}{})
	fake.allMutex.Unlock()
	if fake.AllStub != nil {
		return fake.AllStub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.allReturns.result1, fake.allReturns.result2
}

func (fake *Store) AllCallCount() int {
	fake.allMutex.RLock()
	defer fake.allMutex.RUnlock()
	return len(fake.allArgsForCall)
}

func (fake *Store) AllReturns(result1 []models.Policy, result2 error) {
	fake.AllStub = nil
	fake.allReturns = struct {
		result1 []models.Policy
		result2 error
	}{result1, result2}
}

func (fake *Store) AllReturnsOnCall(i int, result1 []models.Policy, result2 error) {
	fake.AllStub = nil
	if fake.allReturnsOnCall == nil {
		fake.allReturnsOnCall = make(map[int]struct {
			result1 []models.Policy
			result2 error
		})
	}
	fake.allReturnsOnCall[i] = struct {
		result1 []models.Policy
		result2 error
	}{result1, result2}
}

func (fake *Store) Delete(arg1 []models.Policy) error {
	var arg1Copy []models.Policy
	if arg1 != nil {
		arg1Copy = make([]models.Policy, len(arg1))
		copy(arg1Copy, arg1)
	}
	fake.deleteMutex.Lock()
	ret, specificReturn := fake.deleteReturnsOnCall[len(fake.deleteArgsForCall)]
	fake.deleteArgsForCall = append(fake.deleteArgsForCall, struct {
		arg1 []models.Policy
	}{arg1Copy})
	fake.recordInvocation("Delete", []interface{}{arg1Copy})
	fake.deleteMutex.Unlock()
	if fake.DeleteStub != nil {
		return fake.DeleteStub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.deleteReturns.result1
}

func (fake *Store) DeleteCallCount() int {
	fake.deleteMutex.RLock()
	defer fake.deleteMutex.RUnlock()
	return len(fake.deleteArgsForCall)
}

func (fake *Store) DeleteArgsForCall(i int) []models.Policy {
	fake.deleteMutex.RLock()
	defer fake.deleteMutex.RUnlock()
	return fake.deleteArgsForCall[i].arg1
}

func (fake *Store) DeleteReturns(result1 error) {
	fake.DeleteStub = nil
	fake.deleteReturns = struct {
		result1 error
	}{result1}
}

func (fake *Store) DeleteReturnsOnCall(i int, result1 error) {
	fake.DeleteStub = nil
	if fake.deleteReturnsOnCall == nil {
		fake.deleteReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.deleteReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *Store) Tags() ([]models.Tag, error) {
	fake.tagsMutex.Lock()
	ret, specificReturn := fake.tagsReturnsOnCall[len(fake.tagsArgsForCall)]
	fake.tagsArgsForCall = append(fake.tagsArgsForCall, struct{}{})
	fake.recordInvocation("Tags", []interface{}{})
	fake.tagsMutex.Unlock()
	if fake.TagsStub != nil {
		return fake.TagsStub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.tagsReturns.result1, fake.tagsReturns.result2
}

func (fake *Store) TagsCallCount() int {
	fake.tagsMutex.RLock()
	defer fake.tagsMutex.RUnlock()
	return len(fake.tagsArgsForCall)
}

func (fake *Store) TagsReturns(result1 []models.Tag, result2 error) {
	fake.TagsStub = nil
	fake.tagsReturns = struct {
		result1 []models.Tag
		result2 error
	}{result1, result2}
}

func (fake *Store) TagsReturnsOnCall(i int, result1 []models.Tag, result2 error) {
	fake.TagsStub = nil
	if fake.tagsReturnsOnCall == nil {
		fake.tagsReturnsOnCall = make(map[int]struct {
			result1 []models.Tag
			result2 error
		})
	}
	fake.tagsReturnsOnCall[i] = struct {
		result1 []models.Tag
		result2 error
	}{result1, result2}
}

func (fake *Store) ByGuids(arg1 []string, arg2 []string) ([]models.Policy, error) {
	var arg1Copy []string
	if arg1 != nil {
		arg1Copy = make([]string, len(arg1))
		copy(arg1Copy, arg1)
	}
	var arg2Copy []string
	if arg2 != nil {
		arg2Copy = make([]string, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.byGuidsMutex.Lock()
	ret, specificReturn := fake.byGuidsReturnsOnCall[len(fake.byGuidsArgsForCall)]
	fake.byGuidsArgsForCall = append(fake.byGuidsArgsForCall, struct {
		arg1 []string
		arg2 []string
	}{arg1Copy, arg2Copy})
	fake.recordInvocation("ByGuids", []interface{}{arg1Copy, arg2Copy})
	fake.byGuidsMutex.Unlock()
	if fake.ByGuidsStub != nil {
		return fake.ByGuidsStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.byGuidsReturns.result1, fake.byGuidsReturns.result2
}

func (fake *Store) ByGuidsCallCount() int {
	fake.byGuidsMutex.RLock()
	defer fake.byGuidsMutex.RUnlock()
	return len(fake.byGuidsArgsForCall)
}

func (fake *Store) ByGuidsArgsForCall(i int) ([]string, []string) {
	fake.byGuidsMutex.RLock()
	defer fake.byGuidsMutex.RUnlock()
	return fake.byGuidsArgsForCall[i].arg1, fake.byGuidsArgsForCall[i].arg2
}

func (fake *Store) ByGuidsReturns(result1 []models.Policy, result2 error) {
	fake.ByGuidsStub = nil
	fake.byGuidsReturns = struct {
		result1 []models.Policy
		result2 error
	}{result1, result2}
}

func (fake *Store) ByGuidsReturnsOnCall(i int, result1 []models.Policy, result2 error) {
	fake.ByGuidsStub = nil
	if fake.byGuidsReturnsOnCall == nil {
		fake.byGuidsReturnsOnCall = make(map[int]struct {
			result1 []models.Policy
			result2 error
		})
	}
	fake.byGuidsReturnsOnCall[i] = struct {
		result1 []models.Policy
		result2 error
	}{result1, result2}
}

func (fake *Store) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	fake.allMutex.RLock()
	defer fake.allMutex.RUnlock()
	fake.deleteMutex.RLock()
	defer fake.deleteMutex.RUnlock()
	fake.tagsMutex.RLock()
	defer fake.tagsMutex.RUnlock()
	fake.byGuidsMutex.RLock()
	defer fake.byGuidsMutex.RUnlock()
	return fake.invocations
}

func (fake *Store) recordInvocation(key string, args []interface{}) {
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

var _ store.Store = new(Store)
