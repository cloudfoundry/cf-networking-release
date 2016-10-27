package lib

import (
	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/types"
)

//go:generate counterfeiter -o ../fakes/delegator.go --fake-name Delegator . Delegator
type Delegator interface {
	DelegateAdd(delegatePlugin string, netconf []byte) (*types.Result, error)
}

type delegator struct{}

func (*delegator) DelegateAdd(delegatePlugin string, netconf []byte) (*types.Result, error) {
	return invoke.DelegateAdd(delegatePlugin, netconf)
}

func NewDelegator() Delegator { return &delegator{} }
