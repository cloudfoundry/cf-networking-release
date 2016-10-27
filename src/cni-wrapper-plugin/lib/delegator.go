package lib

import (
	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/types"
)

//go:generate counterfeiter -o ../fakes/delegator.go --fake-name Delegator . Delegator
type Delegator interface {
	DelegateAdd(delegatePlugin string, netconf []byte) (*types.Result, error)
	DelegateDel(delegatePlugin string, netconf []byte) error
}

type delegator struct{}

func (*delegator) DelegateAdd(delegatePlugin string, netconf []byte) (*types.Result, error) {
	return invoke.DelegateAdd(delegatePlugin, netconf)
}

func (*delegator) DelegateDel(delegatePlugin string, netconf []byte) error {
	return invoke.DelegateDel(delegatePlugin, netconf)
}

func NewDelegator() Delegator { return &delegator{} }
