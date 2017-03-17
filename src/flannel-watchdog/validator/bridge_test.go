package validator_test

import (
	"errors"
	"flannel-watchdog/validator"
	"flannel-watchdog/validator/fakes"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/vishvananda/netlink"
)

var _ = Describe("Bridge", func() {
	Describe("Validate", func() {
		var (
			logger         lager.Logger
			bridge         *validator.Bridge
			netlinkAdapter *fakes.NetlinkAdapter
		)

		BeforeEach(func() {
			logger = lagertest.NewTestLogger("test")

			netlinkAdapter = &fakes.NetlinkAdapter{}
			netlinkAdapter.LinkByNameReturns(nil, nil)
			addr, err := netlink.ParseAddr("10.255.40.1/24")
			Expect(err).NotTo(HaveOccurred())
			netlinkAdapter.AddrListReturns([]netlink.Addr{*addr}, nil)

			bridge = &validator.Bridge{
				Logger:         logger,
				BridgeName:     "some-bridge-name",
				NetlinkAdapter: netlinkAdapter,
			}
		})

		Context("when a bridge exists and matches the given subnet", func() {
			It("logs that the bridge is found only the first time", func() {
				err := bridge.Validate("10.255.40.1/24")
				Expect(err).NotTo(HaveOccurred())
				Expect(logger).To(gbytes.Say(`Found bridge`))

				err = bridge.Validate("10.255.40.1/24")
				Expect(err).NotTo(HaveOccurred())
				Consistently(logger).ShouldNot(gbytes.Say(`Found bridge`))
			})

			It("successfully exits", func() {
				err := bridge.Validate("10.255.40.1/24")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when a bridge exists and does not match the given subnet", func() {
			It("fails", func() {
				err := bridge.Validate("10.255.50.1/24")
				Expect(err).To(MatchError(`This cell must be restarted (run "bosh restart <job>").  Flannel is out of sync with the local bridge.`))
			})
		})

		Context("when finding the bridge fails", func() {
			BeforeEach(func() {
				netlinkAdapter.LinkByNameReturns(nil, errors.New("banana"))
			})
			It("exits without error and doesn't try to get the address", func() {
				err := bridge.Validate("10.255.50.1/24")
				Expect(err).NotTo(HaveOccurred())

				Expect(netlinkAdapter.AddrListCallCount()).To(Equal(0))
			})
		})

		Context("when getting the bridge ip fails", func() {
			BeforeEach(func() {
				netlinkAdapter.AddrListReturns(nil, errors.New("banana"))
			})

			It("returns an error", func() {
				err := bridge.Validate("10.255.50.1/24")
				Expect(err).To(MatchError("listing addresses: banana"))
			})
		})

		Context("when getting the bridge ip returns more than one address", func() {
			BeforeEach(func() {
				netlinkAdapter.AddrListReturns([]netlink.Addr{netlink.Addr{}, netlink.Addr{}}, nil)
			})

			It("returns an error", func() {
				err := bridge.Validate("10.255.50.1/24")
				Expect(err).To(MatchError("device 'some-bridge-name' does not have one address"))
			})
		})
	})
})
