package converger_test

import (
	"errors"
	"example-apps/reflex/converger"
	"example-apps/reflex/fakes"

	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Converger", func() {
	var (
		listConverger *converger.Converger
		client        *fakes.ReflexClient
		storeWriter   *fakes.StoreWriter
		logger        *lagertest.TestLogger
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		client = &fakes.ReflexClient{}
		client.GetAddressesViaRouterReturns([]string{"1.2.3.4", "4.5.6.7"}, nil)
		client.CheckInstanceStub = func(instance string) bool {
			if instance == "4.5.6.7" {
				return false
			}
			return true
		}

		storeWriter = &fakes.StoreWriter{}

		listConverger = &converger.Converger{
			Logger: logger,
			Client: client,
			Store:  storeWriter,
		}
	})

	It("gets a list of addresses via the router and checks them with the client", func() {
		Expect(listConverger.Converge()).To(Succeed())
		Expect(client.GetAddressesViaRouterCallCount()).To(Equal(1))

		Expect(client.CheckInstanceCallCount()).To(Equal(2))
		var addrs []string
		addrs = append(addrs, client.CheckInstanceArgsForCall(0))
		addrs = append(addrs, client.CheckInstanceArgsForCall(1))
		Expect(addrs).To(ConsistOf([]string{"1.2.3.4", "4.5.6.7"}))
	})

	It("logs the results of CheckInstance", func() {
		Expect(listConverger.Converge()).To(Succeed())

		Expect(logger).To(gbytes.Say("bad.*4.5.6.7.*good.*1.2.3.4"))
	})

	It("adds the verified peers to the store", func() {
		Expect(listConverger.Converge()).To(Succeed())
		Expect(storeWriter.AddCallCount()).To(Equal(1))
		Expect(storeWriter.AddArgsForCall(0)).To(ConsistOf("1.2.3.4"))
	})

	Context("when getting the addresses from the router fails", func() {
		BeforeEach(func() {
			client.GetAddressesViaRouterReturns([]string{}, errors.New("banana"))
		})
		It("logs and returns the error", func() {
			err := listConverger.Converge()
			Expect(logger).To(gbytes.Say("error.*banana"))
			Expect(err).To(MatchError("banana"))
		})
	})
})
