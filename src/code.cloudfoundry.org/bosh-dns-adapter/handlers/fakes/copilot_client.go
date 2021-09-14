// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"sync"

	"code.cloudfoundry.org/bosh-dns-adapter/handlers"
)

type CopilotClient struct {
	IPStub        func(string) (string, error)
	iPMutex       sync.RWMutex
	iPArgsForCall []struct {
		arg1 string
	}
	iPReturns struct {
		result1 string
		result2 error
	}
	iPReturnsOnCall map[int]struct {
		result1 string
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *CopilotClient) IP(arg1 string) (string, error) {
	fake.iPMutex.Lock()
	ret, specificReturn := fake.iPReturnsOnCall[len(fake.iPArgsForCall)]
	fake.iPArgsForCall = append(fake.iPArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.IPStub
	fakeReturns := fake.iPReturns
	fake.recordInvocation("IP", []interface{}{arg1})
	fake.iPMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *CopilotClient) IPCallCount() int {
	fake.iPMutex.RLock()
	defer fake.iPMutex.RUnlock()
	return len(fake.iPArgsForCall)
}

func (fake *CopilotClient) IPCalls(stub func(string) (string, error)) {
	fake.iPMutex.Lock()
	defer fake.iPMutex.Unlock()
	fake.IPStub = stub
}

func (fake *CopilotClient) IPArgsForCall(i int) string {
	fake.iPMutex.RLock()
	defer fake.iPMutex.RUnlock()
	argsForCall := fake.iPArgsForCall[i]
	return argsForCall.arg1
}

func (fake *CopilotClient) IPReturns(result1 string, result2 error) {
	fake.iPMutex.Lock()
	defer fake.iPMutex.Unlock()
	fake.IPStub = nil
	fake.iPReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *CopilotClient) IPReturnsOnCall(i int, result1 string, result2 error) {
	fake.iPMutex.Lock()
	defer fake.iPMutex.Unlock()
	fake.IPStub = nil
	if fake.iPReturnsOnCall == nil {
		fake.iPReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.iPReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *CopilotClient) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.iPMutex.RLock()
	defer fake.iPMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *CopilotClient) recordInvocation(key string, args []interface{}) {
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

var _ handlers.CopilotClient = new(CopilotClient)