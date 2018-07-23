package store_test

import (
	"errors"
	dbfakes "policy-server/db/fakes"
	"policy-server/store"
	"policy-server/store/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EgressPolicyStore", func() {
	var (
		egressPolicyStore *store.EgressPolicyStore
		egressPolicyRepo  *fakes.EgressPolicyRepo

		tx             *dbfakes.Transaction
		egressPolicies []store.EgressPolicy
	)

	BeforeEach(func() {
		egressPolicyRepo = &fakes.EgressPolicyRepo{}
		egressPolicyStore = &store.EgressPolicyStore{
			EgressPolicyRepo: egressPolicyRepo,
		}

		tx = &dbfakes.Transaction{}
		egressPolicies = []store.EgressPolicy{
			{
				Source: store.EgressSource{
					ID: "some-app-guid",
				},
				Destination: store.EgressDestination{
					Protocol: "tcp",
					IPRanges: []store.IPRange{
						{
							Start: "1.2.3.4",
							End:   "1.2.3.5",
						},
					},
				},
			},
			{
				Source: store.EgressSource{
					ID: "different-app-guid",
				},
				Destination: store.EgressDestination{
					Protocol: "udp",
					IPRanges: []store.IPRange{
						{
							Start: "2.2.3.4",
							End:   "2.2.3.5",
						},
					},
				},
			},
		}

		egressPolicyRepo.GetTerminalByAppGUIDReturns(-1, nil)
	})

	Describe("CreateWithTx", func() {
		It("creates a source and destination terminal", func() {
			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateTerminalCallCount()).To(Equal(4))
			Expect(egressPolicyRepo.CreateTerminalArgsForCall(0)).To(Equal(tx))
			Expect(egressPolicyRepo.CreateTerminalArgsForCall(1)).To(Equal(tx))
			Expect(egressPolicyRepo.CreateTerminalArgsForCall(2)).To(Equal(tx))
			Expect(egressPolicyRepo.CreateTerminalArgsForCall(3)).To(Equal(tx))
		})

		It("returns an error when CreateTerminal fails", func() {
			egressPolicyRepo.CreateTerminalReturns(-1, errors.New("OMG WHY DID THIS FAIL"))

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).To(MatchError("failed to create source terminal: OMG WHY DID THIS FAIL"))
		})

		It("creates an app with the sourceTerminalID", func() {
			egressPolicyRepo.CreateTerminalReturnsOnCall(0, 42, nil)
			egressPolicyRepo.CreateTerminalReturnsOnCall(2, 24, nil)

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).NotTo(HaveOccurred())

			Expect(egressPolicyRepo.CreateAppCallCount()).To(Equal(2))
			argTx, argSourceTerminalId, argAppGUID := egressPolicyRepo.CreateAppArgsForCall(0)
			Expect(argTx).To(Equal(tx))
			Expect(argSourceTerminalId).To(Equal(int64(42)))
			Expect(argAppGUID).To(Equal("some-app-guid"))

			argTx, argSourceTerminalId, argAppGUID = egressPolicyRepo.CreateAppArgsForCall(1)
			Expect(argTx).To(Equal(tx))
			Expect(argSourceTerminalId).To(Equal(int64(24)))
			Expect(argAppGUID).To(Equal("different-app-guid"))
		})

		It("returns an error when the CreateApp fails", func() {
			egressPolicyRepo.CreateAppReturns(-1, errors.New("OMG WHY DID THIS FAIL"))

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).To(MatchError("failed to create source app: OMG WHY DID THIS FAIL"))
		})

		It("creates an ip range with the destinationTerminalID", func() {
			egressPolicyRepo.CreateTerminalReturnsOnCall(1, 42, nil)
			egressPolicyRepo.CreateTerminalReturnsOnCall(3, 24, nil)

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateIPRangeCallCount()).To(Equal(2))

			argTx, destinationID, startIP, endIP, protocol := egressPolicyRepo.CreateIPRangeArgsForCall(0)
			Expect(argTx).To(Equal(tx))
			Expect(destinationID).To(Equal(int64(42)))
			Expect(startIP).To(Equal("1.2.3.4"))
			Expect(endIP).To(Equal("1.2.3.5"))
			Expect(protocol).To(Equal("tcp"))

			argTx, destinationID, startIP, endIP, protocol = egressPolicyRepo.CreateIPRangeArgsForCall(1)
			Expect(argTx).To(Equal(tx))
			Expect(destinationID).To(Equal(int64(24)))
			Expect(startIP).To(Equal("2.2.3.4"))
			Expect(endIP).To(Equal("2.2.3.5"))
			Expect(protocol).To(Equal("udp"))
		})

		It("returns an error when the CreateIPRange fails", func() {
			egressPolicyRepo.CreateIPRangeReturns(-1, errors.New("OMG WHY DID THIS FAIL"))

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).To(MatchError("failed to create ip range: OMG WHY DID THIS FAIL"))
		})

		It("creates an egress policy with the right IDs", func() {
			egressPolicyRepo.CreateTerminalReturnsOnCall(0, 11, nil)
			egressPolicyRepo.CreateTerminalReturnsOnCall(1, 22, nil)
			egressPolicyRepo.CreateTerminalReturnsOnCall(2, 33, nil)
			egressPolicyRepo.CreateTerminalReturnsOnCall(3, 44, nil)

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateEgressPolicyCallCount()).To(Equal(2))

			argTx, sourceID, destinationID := egressPolicyRepo.CreateEgressPolicyArgsForCall(0)
			Expect(argTx).To(Equal(tx))
			Expect(sourceID).To(Equal(int64(11)))
			Expect(destinationID).To(Equal(int64(22)))

			argTx, sourceID, destinationID = egressPolicyRepo.CreateEgressPolicyArgsForCall(1)
			Expect(argTx).To(Equal(tx))
			Expect(sourceID).To(Equal(int64(33)))
			Expect(destinationID).To(Equal(int64(44)))
		})

		It("returns an error when the CreateEgressPolicy fails", func() {
			egressPolicyRepo.CreateEgressPolicyReturns(-1, errors.New("OMG WHY DID THIS FAIL"))

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).To(MatchError("failed to create egress policy: OMG WHY DID THIS FAIL"))
		})

		It("uses the existing app terminal id when it exists", func() {
			egressPolicyRepo.GetTerminalByAppGUIDReturns(66, nil)

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateAppCallCount()).To(Equal(0))
			_, sourceID, _ := egressPolicyRepo.CreateEgressPolicyArgsForCall(0)
			Expect(sourceID).To(Equal(int64(66)))
		})

		It("returns an error when the GetTerminalByAppGUID fails", func() {
			egressPolicyRepo.GetTerminalByAppGUIDReturns(-1, errors.New("OMG WHY DID THIS FAIL"))

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).To(MatchError("failed to get terminal by app guid: OMG WHY DID THIS FAIL"))
		})
	})

	Describe("All", func() {
		Context("when there are policies created", func() {
			BeforeEach(func() {
				egressPolicyRepo.GetAllPoliciesReturns(egressPolicies, nil)
			})

			It("should return a list of all policies", func() {
				policies, err := egressPolicyStore.All()
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(Equal(egressPolicies))
			})
		})
	})

	Describe("ByGuids", func() {
		Context("when called with ids", func() {
			BeforeEach(func() {
				egressPolicyRepo.GetByGuidsReturns(egressPolicies, nil)
			})

			It("calls egressPolicyRepo.GetByGuid", func() {
				policies, err := egressPolicyStore.ByGuids([]string{"meow"})
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(Equal(egressPolicies))

				ids := egressPolicyRepo.GetByGuidsArgsForCall(0)
				Expect(ids).To(Equal([]string{"meow"}))
			})
		})

		Context("when an error is returned from the repo", func() {
			BeforeEach(func() {
				egressPolicyRepo.GetByGuidsReturns(nil, errors.New("bark bark"))
			})

			It("calls egressPolicyRepo.GetByGuid", func() {
				_, err := egressPolicyStore.ByGuids([]string{"meow"})
				Expect(err).To(MatchError("failed to get policies by guids: bark bark"))
			})
		})
	})
})
