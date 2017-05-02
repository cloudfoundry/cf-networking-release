package discover_test

import (
	"cni-wrapper-plugin/discover"
	"cni-wrapper-plugin/discover/fakes"
	"errors"
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
)

var _ = Describe("DefaultInterface", func() {
	var (
		netlinkAdapter *fakes.NetlinkAdapter
		netAdapter     *fakes.NetAdapter
		iface          *discover.DefaultInterface
	)

	BeforeEach(func() {
		netlinkAdapter = &fakes.NetlinkAdapter{}
		netAdapter = &fakes.NetAdapter{}
		iface = &discover.DefaultInterface{
			NetlinkAdapter: netlinkAdapter,
			NetAdapter:     netAdapter,
		}

		_, dstNet, _ := net.ParseCIDR("10.212.45.0/24")
		netlinkAdapter.RouteListReturns([]netlink.Route{
			netlink.Route{
				LinkIndex: 42,
				Dst:       nil,
			},
			netlink.Route{
				LinkIndex: 43,
				Dst:       dstNet,
			},
		}, nil)

		netAdapter.InterfaceByIndexReturns(&net.Interface{
			Name: "some-interface",
		}, nil)
	})

	It("returns the name of the default interface", func() {
		name, err := iface.Name()
		Expect(err).NotTo(HaveOccurred())

		Expect(netlinkAdapter.RouteListCallCount()).To(Equal(1))
		link, family := netlinkAdapter.RouteListArgsForCall(0)
		Expect(link).To(BeNil())
		Expect(family).To(Equal(netlink.FAMILY_V4))

		Expect(netAdapter.InterfaceByIndexCallCount()).To(Equal(1))
		Expect(netAdapter.InterfaceByIndexArgsForCall(0)).To(Equal(42))

		Expect(name).To(Equal("some-interface"))
	})

	Context("when NetlinkAdapter returns an error", func() {
		BeforeEach(func() {
			netlinkAdapter.RouteListReturns(nil, errors.New("apple"))
		})
		It("returns a sensible error", func() {
			_, err := iface.Name()
			Expect(err).To(MatchError("route list: apple"))
		})
	})

	Context("when NetAdapter returns an error", func() {
		BeforeEach(func() {
			netAdapter.InterfaceByIndexReturns(nil, errors.New("cherry"))
		})
		It("returns a sensible error", func() {
			_, err := iface.Name()
			Expect(err).To(MatchError("interface by index: cherry"))
		})
	})

	Context("when no route with an empty Dst exists", func() {
		BeforeEach(func() {
			_, dstNet, _ := net.ParseCIDR("10.212.45.0/24")
			netlinkAdapter.RouteListReturns([]netlink.Route{
				netlink.Route{
					LinkIndex: 42,
					Dst:       dstNet,
				},
				netlink.Route{
					LinkIndex: 43,
					Dst:       dstNet,
				},
			}, nil)
		})
		It("returns a sensible error", func() {
			_, err := iface.Name()
			Expect(err).To(MatchError("no default route"))
		})
	})

	Context("when multiple routes with an empty Dst exist", func() {
		BeforeEach(func() {
			netlinkAdapter.RouteListReturns([]netlink.Route{
				netlink.Route{
					LinkIndex: 42,
					Dst:       nil,
				},
				netlink.Route{
					LinkIndex: 43,
					Dst:       nil,
				},
			}, nil)
		})
		It("returns a sensible error", func() {
			_, err := iface.Name()
			Expect(err).To(MatchError("multiple possible default routes"))
		})
	})

})
