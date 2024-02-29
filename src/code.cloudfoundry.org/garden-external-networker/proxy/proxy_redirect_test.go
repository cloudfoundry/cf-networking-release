package proxy_test

import (
	"errors"
	"strconv"

	"code.cloudfoundry.org/garden-external-networker/fakes"
	"code.cloudfoundry.org/garden-external-networker/proxy"
	lib_fakes "code.cloudfoundry.org/lib/fakes"
	"code.cloudfoundry.org/lib/rules"

	"github.com/containernetworking/plugins/pkg/ns"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

//lint:ignore U1000 used in fakes
//go:generate counterfeiter -o ../fakes/netNS.go --fake-name NetNS . netNS
type netNS interface {
	ns.NetNS
}

var _ = Describe("Redirect", func() {
	var (
		proxyRedirect    *proxy.Redirect
		iptablesAdapter  *lib_fakes.IPTablesAdapter
		namespaceAdapter *fakes.NamespaceAdapter
		netNS            *fakes.NetNS

		containerNetNamespace string
		redirectCIDR          string
		proxyPort             int
		proxyUID              int
	)

	BeforeEach(func() {
		iptablesAdapter = &lib_fakes.IPTablesAdapter{}
		namespaceAdapter = &fakes.NamespaceAdapter{}
		netNS = &fakes.NetNS{}
		netNS.DoStub = func(toRun func(ns.NetNS) error) error {
			return toRun(netNS)
		}

		namespaceAdapter.GetNSReturns(netNS, nil)

		containerNetNamespace = "some-network-namespace"
		redirectCIDR = "10.255.0.0/24"
		proxyPort = 1111
		proxyUID = 1

		proxyRedirect = &proxy.Redirect{
			IPTables:                   iptablesAdapter,
			NamespaceAdapter:           namespaceAdapter,
			RedirectCIDR:               redirectCIDR,
			ProxyPort:                  proxyPort,
			ProxyUID:                   proxyUID,
			EnableIngressProxyRedirect: true,
		}
	})

	Describe("Apply", func() {
		It("apply iptables rules to redirect traffic to the proxy in the container net namespace", func() {
			err := proxyRedirect.Apply(containerNetNamespace)
			Expect(err).NotTo(HaveOccurred())

			Expect(namespaceAdapter.GetNSCallCount()).To(Equal(1))
			Expect(namespaceAdapter.GetNSArgsForCall(0)).To(Equal(containerNetNamespace))

			Expect(netNS.DoCallCount()).To(Equal(1))

			Expect(iptablesAdapter.BulkAppendCallCount()).To(Equal(2))
			table, name, iptablesRules := iptablesAdapter.BulkAppendArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(name).To(Equal("OUTPUT"))
			Expect(iptablesRules).To(Equal([]rules.IPTablesRule{
				{
					"-d", redirectCIDR,
					"-p", "tcp",
					"-j", "REDIRECT", "--to-port", string(strconv.Itoa(proxyPort)),
				},
			}))

			table, name, iptablesRules = iptablesAdapter.BulkAppendArgsForCall(1)
			Expect(table).To(Equal("nat"))
			Expect(name).To(Equal("PREROUTING"))
			Expect(iptablesRules).To(Equal([]rules.IPTablesRule{
				{
					"-p", "tcp",
					"-j", "REDIRECT", "--to-port", string(strconv.Itoa(proxyPort)),
				},
			}))
		})

		Context("when bulk appending to OUTPUT fails", func() {
			BeforeEach(func() {
				iptablesAdapter.BulkAppendReturns(errors.New("banana"))
			})

			It("returns an error", func() {
				err := proxyRedirect.Apply(containerNetNamespace)
				Expect(err).To(MatchError("do in container: banana"))
			})
		})

		Context("when the redirect cidr is empty", func() {
			BeforeEach(func() {
				proxyRedirect.RedirectCIDR = ""
			})

			It("doesn't write the redirect cidr rule", func() {
				Expect(proxyRedirect.Apply(containerNetNamespace)).To(Succeed())
				Expect(netNS.DoCallCount()).To(Equal(1))
				table, name, iptablesRules := iptablesAdapter.BulkAppendArgsForCall(0)
				Expect(table).To(Equal("nat"))
				Expect(name).To(Equal("PREROUTING"))
				Expect(iptablesRules).To(Equal([]rules.IPTablesRule{
					{
						"-p", "tcp",
						"-j", "REDIRECT", "--to-port", string(strconv.Itoa(proxyPort)),
					},
				}))
			})
		})

		Context("when enable ingress proxy redirect is set to false", func() {
			BeforeEach(func() {
				proxyRedirect.EnableIngressProxyRedirect = false
			})

			It("does not write the ingress proxy redirect rule", func() {
				Expect(proxyRedirect.Apply(containerNetNamespace)).To(Succeed())
				Expect(netNS.DoCallCount()).To(Equal(1))
				table, name, iptablesRules := iptablesAdapter.BulkAppendArgsForCall(0)
				Expect(table).To(Equal("nat"))
				Expect(name).To(Equal("OUTPUT"))
				Expect(iptablesRules).To(Equal([]rules.IPTablesRule{
					{
						"-d", redirectCIDR,
						"-p", "tcp",
						"-j", "REDIRECT", "--to-port", string(strconv.Itoa(proxyPort)),
					},
				}))
			})
		})
	})
})
