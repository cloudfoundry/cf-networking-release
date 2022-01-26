// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"sync"

	"code.cloudfoundry.org/lib/policy_client"
	"code.cloudfoundry.org/policy-server/api"
	"code.cloudfoundry.org/policy-server/api/api_v0"
)

type ExternalPolicyClient struct {
	AddPoliciesStub        func(string, []api.Policy) error
	addPoliciesMutex       sync.RWMutex
	addPoliciesArgsForCall []struct {
		arg1 string
		arg2 []api.Policy
	}
	addPoliciesReturns struct {
		result1 error
	}
	addPoliciesReturnsOnCall map[int]struct {
		result1 error
	}
	AddPoliciesV0Stub        func(string, []api_v0.Policy) error
	addPoliciesV0Mutex       sync.RWMutex
	addPoliciesV0ArgsForCall []struct {
		arg1 string
		arg2 []api_v0.Policy
	}
	addPoliciesV0Returns struct {
		result1 error
	}
	addPoliciesV0ReturnsOnCall map[int]struct {
		result1 error
	}
	DeletePoliciesStub        func(string, []api.Policy) error
	deletePoliciesMutex       sync.RWMutex
	deletePoliciesArgsForCall []struct {
		arg1 string
		arg2 []api.Policy
	}
	deletePoliciesReturns struct {
		result1 error
	}
	deletePoliciesReturnsOnCall map[int]struct {
		result1 error
	}
	DeletePoliciesV0Stub        func(string, []api_v0.Policy) error
	deletePoliciesV0Mutex       sync.RWMutex
	deletePoliciesV0ArgsForCall []struct {
		arg1 string
		arg2 []api_v0.Policy
	}
	deletePoliciesV0Returns struct {
		result1 error
	}
	deletePoliciesV0ReturnsOnCall map[int]struct {
		result1 error
	}
	GetPoliciesStub        func(string) ([]api.Policy, error)
	getPoliciesMutex       sync.RWMutex
	getPoliciesArgsForCall []struct {
		arg1 string
	}
	getPoliciesReturns struct {
		result1 []api.Policy
		result2 error
	}
	getPoliciesReturnsOnCall map[int]struct {
		result1 []api.Policy
		result2 error
	}
	GetPoliciesByIDStub        func(string, ...string) ([]api.Policy, error)
	getPoliciesByIDMutex       sync.RWMutex
	getPoliciesByIDArgsForCall []struct {
		arg1 string
		arg2 []string
	}
	getPoliciesByIDReturns struct {
		result1 []api.Policy
		result2 error
	}
	getPoliciesByIDReturnsOnCall map[int]struct {
		result1 []api.Policy
		result2 error
	}
	GetPoliciesV0Stub        func(string) ([]api_v0.Policy, error)
	getPoliciesV0Mutex       sync.RWMutex
	getPoliciesV0ArgsForCall []struct {
		arg1 string
	}
	getPoliciesV0Returns struct {
		result1 []api_v0.Policy
		result2 error
	}
	getPoliciesV0ReturnsOnCall map[int]struct {
		result1 []api_v0.Policy
		result2 error
	}
	GetPoliciesV0ByIDStub        func(string, ...string) ([]api_v0.Policy, error)
	getPoliciesV0ByIDMutex       sync.RWMutex
	getPoliciesV0ByIDArgsForCall []struct {
		arg1 string
		arg2 []string
	}
	getPoliciesV0ByIDReturns struct {
		result1 []api_v0.Policy
		result2 error
	}
	getPoliciesV0ByIDReturnsOnCall map[int]struct {
		result1 []api_v0.Policy
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *ExternalPolicyClient) AddPolicies(arg1 string, arg2 []api.Policy) error {
	var arg2Copy []api.Policy
	if arg2 != nil {
		arg2Copy = make([]api.Policy, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.addPoliciesMutex.Lock()
	ret, specificReturn := fake.addPoliciesReturnsOnCall[len(fake.addPoliciesArgsForCall)]
	fake.addPoliciesArgsForCall = append(fake.addPoliciesArgsForCall, struct {
		arg1 string
		arg2 []api.Policy
	}{arg1, arg2Copy})
	stub := fake.AddPoliciesStub
	fakeReturns := fake.addPoliciesReturns
	fake.recordInvocation("AddPolicies", []interface{}{arg1, arg2Copy})
	fake.addPoliciesMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *ExternalPolicyClient) AddPoliciesCallCount() int {
	fake.addPoliciesMutex.RLock()
	defer fake.addPoliciesMutex.RUnlock()
	return len(fake.addPoliciesArgsForCall)
}

func (fake *ExternalPolicyClient) AddPoliciesCalls(stub func(string, []api.Policy) error) {
	fake.addPoliciesMutex.Lock()
	defer fake.addPoliciesMutex.Unlock()
	fake.AddPoliciesStub = stub
}

func (fake *ExternalPolicyClient) AddPoliciesArgsForCall(i int) (string, []api.Policy) {
	fake.addPoliciesMutex.RLock()
	defer fake.addPoliciesMutex.RUnlock()
	argsForCall := fake.addPoliciesArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *ExternalPolicyClient) AddPoliciesReturns(result1 error) {
	fake.addPoliciesMutex.Lock()
	defer fake.addPoliciesMutex.Unlock()
	fake.AddPoliciesStub = nil
	fake.addPoliciesReturns = struct {
		result1 error
	}{result1}
}

func (fake *ExternalPolicyClient) AddPoliciesReturnsOnCall(i int, result1 error) {
	fake.addPoliciesMutex.Lock()
	defer fake.addPoliciesMutex.Unlock()
	fake.AddPoliciesStub = nil
	if fake.addPoliciesReturnsOnCall == nil {
		fake.addPoliciesReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.addPoliciesReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *ExternalPolicyClient) AddPoliciesV0(arg1 string, arg2 []api_v0.Policy) error {
	var arg2Copy []api_v0.Policy
	if arg2 != nil {
		arg2Copy = make([]api_v0.Policy, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.addPoliciesV0Mutex.Lock()
	ret, specificReturn := fake.addPoliciesV0ReturnsOnCall[len(fake.addPoliciesV0ArgsForCall)]
	fake.addPoliciesV0ArgsForCall = append(fake.addPoliciesV0ArgsForCall, struct {
		arg1 string
		arg2 []api_v0.Policy
	}{arg1, arg2Copy})
	stub := fake.AddPoliciesV0Stub
	fakeReturns := fake.addPoliciesV0Returns
	fake.recordInvocation("AddPoliciesV0", []interface{}{arg1, arg2Copy})
	fake.addPoliciesV0Mutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *ExternalPolicyClient) AddPoliciesV0CallCount() int {
	fake.addPoliciesV0Mutex.RLock()
	defer fake.addPoliciesV0Mutex.RUnlock()
	return len(fake.addPoliciesV0ArgsForCall)
}

func (fake *ExternalPolicyClient) AddPoliciesV0Calls(stub func(string, []api_v0.Policy) error) {
	fake.addPoliciesV0Mutex.Lock()
	defer fake.addPoliciesV0Mutex.Unlock()
	fake.AddPoliciesV0Stub = stub
}

func (fake *ExternalPolicyClient) AddPoliciesV0ArgsForCall(i int) (string, []api_v0.Policy) {
	fake.addPoliciesV0Mutex.RLock()
	defer fake.addPoliciesV0Mutex.RUnlock()
	argsForCall := fake.addPoliciesV0ArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *ExternalPolicyClient) AddPoliciesV0Returns(result1 error) {
	fake.addPoliciesV0Mutex.Lock()
	defer fake.addPoliciesV0Mutex.Unlock()
	fake.AddPoliciesV0Stub = nil
	fake.addPoliciesV0Returns = struct {
		result1 error
	}{result1}
}

func (fake *ExternalPolicyClient) AddPoliciesV0ReturnsOnCall(i int, result1 error) {
	fake.addPoliciesV0Mutex.Lock()
	defer fake.addPoliciesV0Mutex.Unlock()
	fake.AddPoliciesV0Stub = nil
	if fake.addPoliciesV0ReturnsOnCall == nil {
		fake.addPoliciesV0ReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.addPoliciesV0ReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *ExternalPolicyClient) DeletePolicies(arg1 string, arg2 []api.Policy) error {
	var arg2Copy []api.Policy
	if arg2 != nil {
		arg2Copy = make([]api.Policy, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.deletePoliciesMutex.Lock()
	ret, specificReturn := fake.deletePoliciesReturnsOnCall[len(fake.deletePoliciesArgsForCall)]
	fake.deletePoliciesArgsForCall = append(fake.deletePoliciesArgsForCall, struct {
		arg1 string
		arg2 []api.Policy
	}{arg1, arg2Copy})
	stub := fake.DeletePoliciesStub
	fakeReturns := fake.deletePoliciesReturns
	fake.recordInvocation("DeletePolicies", []interface{}{arg1, arg2Copy})
	fake.deletePoliciesMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *ExternalPolicyClient) DeletePoliciesCallCount() int {
	fake.deletePoliciesMutex.RLock()
	defer fake.deletePoliciesMutex.RUnlock()
	return len(fake.deletePoliciesArgsForCall)
}

func (fake *ExternalPolicyClient) DeletePoliciesCalls(stub func(string, []api.Policy) error) {
	fake.deletePoliciesMutex.Lock()
	defer fake.deletePoliciesMutex.Unlock()
	fake.DeletePoliciesStub = stub
}

func (fake *ExternalPolicyClient) DeletePoliciesArgsForCall(i int) (string, []api.Policy) {
	fake.deletePoliciesMutex.RLock()
	defer fake.deletePoliciesMutex.RUnlock()
	argsForCall := fake.deletePoliciesArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *ExternalPolicyClient) DeletePoliciesReturns(result1 error) {
	fake.deletePoliciesMutex.Lock()
	defer fake.deletePoliciesMutex.Unlock()
	fake.DeletePoliciesStub = nil
	fake.deletePoliciesReturns = struct {
		result1 error
	}{result1}
}

func (fake *ExternalPolicyClient) DeletePoliciesReturnsOnCall(i int, result1 error) {
	fake.deletePoliciesMutex.Lock()
	defer fake.deletePoliciesMutex.Unlock()
	fake.DeletePoliciesStub = nil
	if fake.deletePoliciesReturnsOnCall == nil {
		fake.deletePoliciesReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.deletePoliciesReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *ExternalPolicyClient) DeletePoliciesV0(arg1 string, arg2 []api_v0.Policy) error {
	var arg2Copy []api_v0.Policy
	if arg2 != nil {
		arg2Copy = make([]api_v0.Policy, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.deletePoliciesV0Mutex.Lock()
	ret, specificReturn := fake.deletePoliciesV0ReturnsOnCall[len(fake.deletePoliciesV0ArgsForCall)]
	fake.deletePoliciesV0ArgsForCall = append(fake.deletePoliciesV0ArgsForCall, struct {
		arg1 string
		arg2 []api_v0.Policy
	}{arg1, arg2Copy})
	stub := fake.DeletePoliciesV0Stub
	fakeReturns := fake.deletePoliciesV0Returns
	fake.recordInvocation("DeletePoliciesV0", []interface{}{arg1, arg2Copy})
	fake.deletePoliciesV0Mutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *ExternalPolicyClient) DeletePoliciesV0CallCount() int {
	fake.deletePoliciesV0Mutex.RLock()
	defer fake.deletePoliciesV0Mutex.RUnlock()
	return len(fake.deletePoliciesV0ArgsForCall)
}

func (fake *ExternalPolicyClient) DeletePoliciesV0Calls(stub func(string, []api_v0.Policy) error) {
	fake.deletePoliciesV0Mutex.Lock()
	defer fake.deletePoliciesV0Mutex.Unlock()
	fake.DeletePoliciesV0Stub = stub
}

func (fake *ExternalPolicyClient) DeletePoliciesV0ArgsForCall(i int) (string, []api_v0.Policy) {
	fake.deletePoliciesV0Mutex.RLock()
	defer fake.deletePoliciesV0Mutex.RUnlock()
	argsForCall := fake.deletePoliciesV0ArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *ExternalPolicyClient) DeletePoliciesV0Returns(result1 error) {
	fake.deletePoliciesV0Mutex.Lock()
	defer fake.deletePoliciesV0Mutex.Unlock()
	fake.DeletePoliciesV0Stub = nil
	fake.deletePoliciesV0Returns = struct {
		result1 error
	}{result1}
}

func (fake *ExternalPolicyClient) DeletePoliciesV0ReturnsOnCall(i int, result1 error) {
	fake.deletePoliciesV0Mutex.Lock()
	defer fake.deletePoliciesV0Mutex.Unlock()
	fake.DeletePoliciesV0Stub = nil
	if fake.deletePoliciesV0ReturnsOnCall == nil {
		fake.deletePoliciesV0ReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.deletePoliciesV0ReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *ExternalPolicyClient) GetPolicies(arg1 string) ([]api.Policy, error) {
	fake.getPoliciesMutex.Lock()
	ret, specificReturn := fake.getPoliciesReturnsOnCall[len(fake.getPoliciesArgsForCall)]
	fake.getPoliciesArgsForCall = append(fake.getPoliciesArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.GetPoliciesStub
	fakeReturns := fake.getPoliciesReturns
	fake.recordInvocation("GetPolicies", []interface{}{arg1})
	fake.getPoliciesMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *ExternalPolicyClient) GetPoliciesCallCount() int {
	fake.getPoliciesMutex.RLock()
	defer fake.getPoliciesMutex.RUnlock()
	return len(fake.getPoliciesArgsForCall)
}

func (fake *ExternalPolicyClient) GetPoliciesCalls(stub func(string) ([]api.Policy, error)) {
	fake.getPoliciesMutex.Lock()
	defer fake.getPoliciesMutex.Unlock()
	fake.GetPoliciesStub = stub
}

func (fake *ExternalPolicyClient) GetPoliciesArgsForCall(i int) string {
	fake.getPoliciesMutex.RLock()
	defer fake.getPoliciesMutex.RUnlock()
	argsForCall := fake.getPoliciesArgsForCall[i]
	return argsForCall.arg1
}

func (fake *ExternalPolicyClient) GetPoliciesReturns(result1 []api.Policy, result2 error) {
	fake.getPoliciesMutex.Lock()
	defer fake.getPoliciesMutex.Unlock()
	fake.GetPoliciesStub = nil
	fake.getPoliciesReturns = struct {
		result1 []api.Policy
		result2 error
	}{result1, result2}
}

func (fake *ExternalPolicyClient) GetPoliciesReturnsOnCall(i int, result1 []api.Policy, result2 error) {
	fake.getPoliciesMutex.Lock()
	defer fake.getPoliciesMutex.Unlock()
	fake.GetPoliciesStub = nil
	if fake.getPoliciesReturnsOnCall == nil {
		fake.getPoliciesReturnsOnCall = make(map[int]struct {
			result1 []api.Policy
			result2 error
		})
	}
	fake.getPoliciesReturnsOnCall[i] = struct {
		result1 []api.Policy
		result2 error
	}{result1, result2}
}

func (fake *ExternalPolicyClient) GetPoliciesByID(arg1 string, arg2 ...string) ([]api.Policy, error) {
	fake.getPoliciesByIDMutex.Lock()
	ret, specificReturn := fake.getPoliciesByIDReturnsOnCall[len(fake.getPoliciesByIDArgsForCall)]
	fake.getPoliciesByIDArgsForCall = append(fake.getPoliciesByIDArgsForCall, struct {
		arg1 string
		arg2 []string
	}{arg1, arg2})
	stub := fake.GetPoliciesByIDStub
	fakeReturns := fake.getPoliciesByIDReturns
	fake.recordInvocation("GetPoliciesByID", []interface{}{arg1, arg2})
	fake.getPoliciesByIDMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *ExternalPolicyClient) GetPoliciesByIDCallCount() int {
	fake.getPoliciesByIDMutex.RLock()
	defer fake.getPoliciesByIDMutex.RUnlock()
	return len(fake.getPoliciesByIDArgsForCall)
}

func (fake *ExternalPolicyClient) GetPoliciesByIDCalls(stub func(string, ...string) ([]api.Policy, error)) {
	fake.getPoliciesByIDMutex.Lock()
	defer fake.getPoliciesByIDMutex.Unlock()
	fake.GetPoliciesByIDStub = stub
}

func (fake *ExternalPolicyClient) GetPoliciesByIDArgsForCall(i int) (string, []string) {
	fake.getPoliciesByIDMutex.RLock()
	defer fake.getPoliciesByIDMutex.RUnlock()
	argsForCall := fake.getPoliciesByIDArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *ExternalPolicyClient) GetPoliciesByIDReturns(result1 []api.Policy, result2 error) {
	fake.getPoliciesByIDMutex.Lock()
	defer fake.getPoliciesByIDMutex.Unlock()
	fake.GetPoliciesByIDStub = nil
	fake.getPoliciesByIDReturns = struct {
		result1 []api.Policy
		result2 error
	}{result1, result2}
}

func (fake *ExternalPolicyClient) GetPoliciesByIDReturnsOnCall(i int, result1 []api.Policy, result2 error) {
	fake.getPoliciesByIDMutex.Lock()
	defer fake.getPoliciesByIDMutex.Unlock()
	fake.GetPoliciesByIDStub = nil
	if fake.getPoliciesByIDReturnsOnCall == nil {
		fake.getPoliciesByIDReturnsOnCall = make(map[int]struct {
			result1 []api.Policy
			result2 error
		})
	}
	fake.getPoliciesByIDReturnsOnCall[i] = struct {
		result1 []api.Policy
		result2 error
	}{result1, result2}
}

func (fake *ExternalPolicyClient) GetPoliciesV0(arg1 string) ([]api_v0.Policy, error) {
	fake.getPoliciesV0Mutex.Lock()
	ret, specificReturn := fake.getPoliciesV0ReturnsOnCall[len(fake.getPoliciesV0ArgsForCall)]
	fake.getPoliciesV0ArgsForCall = append(fake.getPoliciesV0ArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.GetPoliciesV0Stub
	fakeReturns := fake.getPoliciesV0Returns
	fake.recordInvocation("GetPoliciesV0", []interface{}{arg1})
	fake.getPoliciesV0Mutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *ExternalPolicyClient) GetPoliciesV0CallCount() int {
	fake.getPoliciesV0Mutex.RLock()
	defer fake.getPoliciesV0Mutex.RUnlock()
	return len(fake.getPoliciesV0ArgsForCall)
}

func (fake *ExternalPolicyClient) GetPoliciesV0Calls(stub func(string) ([]api_v0.Policy, error)) {
	fake.getPoliciesV0Mutex.Lock()
	defer fake.getPoliciesV0Mutex.Unlock()
	fake.GetPoliciesV0Stub = stub
}

func (fake *ExternalPolicyClient) GetPoliciesV0ArgsForCall(i int) string {
	fake.getPoliciesV0Mutex.RLock()
	defer fake.getPoliciesV0Mutex.RUnlock()
	argsForCall := fake.getPoliciesV0ArgsForCall[i]
	return argsForCall.arg1
}

func (fake *ExternalPolicyClient) GetPoliciesV0Returns(result1 []api_v0.Policy, result2 error) {
	fake.getPoliciesV0Mutex.Lock()
	defer fake.getPoliciesV0Mutex.Unlock()
	fake.GetPoliciesV0Stub = nil
	fake.getPoliciesV0Returns = struct {
		result1 []api_v0.Policy
		result2 error
	}{result1, result2}
}

func (fake *ExternalPolicyClient) GetPoliciesV0ReturnsOnCall(i int, result1 []api_v0.Policy, result2 error) {
	fake.getPoliciesV0Mutex.Lock()
	defer fake.getPoliciesV0Mutex.Unlock()
	fake.GetPoliciesV0Stub = nil
	if fake.getPoliciesV0ReturnsOnCall == nil {
		fake.getPoliciesV0ReturnsOnCall = make(map[int]struct {
			result1 []api_v0.Policy
			result2 error
		})
	}
	fake.getPoliciesV0ReturnsOnCall[i] = struct {
		result1 []api_v0.Policy
		result2 error
	}{result1, result2}
}

func (fake *ExternalPolicyClient) GetPoliciesV0ByID(arg1 string, arg2 ...string) ([]api_v0.Policy, error) {
	fake.getPoliciesV0ByIDMutex.Lock()
	ret, specificReturn := fake.getPoliciesV0ByIDReturnsOnCall[len(fake.getPoliciesV0ByIDArgsForCall)]
	fake.getPoliciesV0ByIDArgsForCall = append(fake.getPoliciesV0ByIDArgsForCall, struct {
		arg1 string
		arg2 []string
	}{arg1, arg2})
	stub := fake.GetPoliciesV0ByIDStub
	fakeReturns := fake.getPoliciesV0ByIDReturns
	fake.recordInvocation("GetPoliciesV0ByID", []interface{}{arg1, arg2})
	fake.getPoliciesV0ByIDMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *ExternalPolicyClient) GetPoliciesV0ByIDCallCount() int {
	fake.getPoliciesV0ByIDMutex.RLock()
	defer fake.getPoliciesV0ByIDMutex.RUnlock()
	return len(fake.getPoliciesV0ByIDArgsForCall)
}

func (fake *ExternalPolicyClient) GetPoliciesV0ByIDCalls(stub func(string, ...string) ([]api_v0.Policy, error)) {
	fake.getPoliciesV0ByIDMutex.Lock()
	defer fake.getPoliciesV0ByIDMutex.Unlock()
	fake.GetPoliciesV0ByIDStub = stub
}

func (fake *ExternalPolicyClient) GetPoliciesV0ByIDArgsForCall(i int) (string, []string) {
	fake.getPoliciesV0ByIDMutex.RLock()
	defer fake.getPoliciesV0ByIDMutex.RUnlock()
	argsForCall := fake.getPoliciesV0ByIDArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *ExternalPolicyClient) GetPoliciesV0ByIDReturns(result1 []api_v0.Policy, result2 error) {
	fake.getPoliciesV0ByIDMutex.Lock()
	defer fake.getPoliciesV0ByIDMutex.Unlock()
	fake.GetPoliciesV0ByIDStub = nil
	fake.getPoliciesV0ByIDReturns = struct {
		result1 []api_v0.Policy
		result2 error
	}{result1, result2}
}

func (fake *ExternalPolicyClient) GetPoliciesV0ByIDReturnsOnCall(i int, result1 []api_v0.Policy, result2 error) {
	fake.getPoliciesV0ByIDMutex.Lock()
	defer fake.getPoliciesV0ByIDMutex.Unlock()
	fake.GetPoliciesV0ByIDStub = nil
	if fake.getPoliciesV0ByIDReturnsOnCall == nil {
		fake.getPoliciesV0ByIDReturnsOnCall = make(map[int]struct {
			result1 []api_v0.Policy
			result2 error
		})
	}
	fake.getPoliciesV0ByIDReturnsOnCall[i] = struct {
		result1 []api_v0.Policy
		result2 error
	}{result1, result2}
}

func (fake *ExternalPolicyClient) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.addPoliciesMutex.RLock()
	defer fake.addPoliciesMutex.RUnlock()
	fake.addPoliciesV0Mutex.RLock()
	defer fake.addPoliciesV0Mutex.RUnlock()
	fake.deletePoliciesMutex.RLock()
	defer fake.deletePoliciesMutex.RUnlock()
	fake.deletePoliciesV0Mutex.RLock()
	defer fake.deletePoliciesV0Mutex.RUnlock()
	fake.getPoliciesMutex.RLock()
	defer fake.getPoliciesMutex.RUnlock()
	fake.getPoliciesByIDMutex.RLock()
	defer fake.getPoliciesByIDMutex.RUnlock()
	fake.getPoliciesV0Mutex.RLock()
	defer fake.getPoliciesV0Mutex.RUnlock()
	fake.getPoliciesV0ByIDMutex.RLock()
	defer fake.getPoliciesV0ByIDMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *ExternalPolicyClient) recordInvocation(key string, args []interface{}) {
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

var _ policy_client.ExternalPolicyClient = new(ExternalPolicyClient)
