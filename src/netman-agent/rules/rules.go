package rules

import (
	"fmt"
	"strconv"

	"github.com/pivotal-golang/lager"
)

type Rule interface {
	Enforce(string) error
	Chain(int64) string
}

type LocalAllowRule struct {
	SrcIP    string
	DstIP    string
	Port     int
	Proto    string
	IPTables iptables
	Logger   lager.Logger
}

func (r LocalAllowRule) Chain(timeStamp int64) string {
	return fmt.Sprintf("netman--forward-%d", timeStamp)
}

func (r LocalAllowRule) Enforce(chain string) error {
	err := r.IPTables.AppendUnique("filter", chain, []string{
		"-i", "cni-flannel0",
		"-s", r.SrcIP,
		"-d", r.DstIP,
		"-p", r.Proto,
		"--dport", strconv.Itoa(r.Port),
		"-j", "ACCEPT",
	}...)
	if err != nil {
		r.Logger.Error("append-rule", err)
		return fmt.Errorf("appending rule: %s", err)
	}

	r.Logger.Info("enforce-local-rule", lager.Data{
		"srcIP": r.SrcIP,
		"dstIP": r.DstIP,
		"port":  r.Port,
		"proto": r.Proto,
	})

	return nil
}

type RemoteAllowRule struct {
	SrcTag   string
	DstIP    string
	Port     int
	Proto    string
	VNI      int
	IPTables iptables
	Logger   lager.Logger
}

func (r RemoteAllowRule) Chain(timeStamp int64) string {
	return ""
}

func (r RemoteAllowRule) Enforce(chain string) error {
	r.Logger.Info("enforce-remote-rule", lager.Data{
		"srcTag": r.SrcTag,
		"dstIP":  r.DstIP,
		"port":   r.Port,
		"proto":  r.Proto,
		"vni":    r.VNI,
	})

	return nil
}

type LocalTagRule struct {
	SourceTag         string
	SourceContainerIP string
	IPTables          iptables
	Logger            lager.Logger
}

func (r LocalTagRule) Chain(timeStamp int64) string {
	return ""
}

func (r LocalTagRule) Enforce(chain string) error {
	r.Logger.Info("set-local-tag", lager.Data{
		"srcTag": r.SourceTag,
		"srcIP":  r.SourceContainerIP,
	})

	return nil
}
