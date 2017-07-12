package merger_test

import (
	"errors"
	"log-transformer/merger"
	"log-transformer/merger/fakes"
	"log-transformer/parser"
	"log-transformer/repository"

	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Merger", func() {
	var (
		logMerger         *merger.Merger
		container         repository.Container
		parsedData        parser.ParsedData
		fakeContainerRepo *fakes.ContainerRepo
	)
	BeforeEach(func() {
		fakeContainerRepo = &fakes.ContainerRepo{}
		logMerger = &merger.Merger{
			ContainerRepo: fakeContainerRepo,
		}

		container = repository.Container{
			Handle:  "some-handle",
			AppID:   "some-app-id",
			SpaceID: "some-space-id",
			OrgID:   "some-org-id",
		}

		parsedData = parser.ParsedData{
			Direction:       "ingress",
			Allowed:         true,
			SourceIP:        "1.2.3.4",
			DestinationIP:   "5.6.7.8",
			SourcePort:      1234,
			DestinationPort: 9999,
			Protocol:        "some-proto",
			Mark:            "some-mark",
			ICMPType:        42,
			ICMPCode:        13,
		}

		fakeContainerRepo.GetByIPReturns(container, nil)
	})

	It("merges the container metadata with the kernel log data", func() {
		merged, err := logMerger.Merge(parsedData)
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeContainerRepo.GetByIPCallCount()).To(Equal(1))
		Expect(fakeContainerRepo.GetByIPArgsForCall(0)).To(Equal("5.6.7.8"))

		Expect(merged).To(Equal(merger.IPTablesLogData{
			Message: "ingress-allowed",
			Data:    lager.Data{"destination": container, "packet": parsedData},
		}))
	})

	Context("when the data is for an egress packet", func() {
		BeforeEach(func() {
			parsedData.Direction = "egress"
		})
		It("merges the data", func() {
			merged, err := logMerger.Merge(parsedData)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeContainerRepo.GetByIPCallCount()).To(Equal(1))
			Expect(fakeContainerRepo.GetByIPArgsForCall(0)).To(Equal("1.2.3.4"))

			Expect(merged).To(Equal(merger.IPTablesLogData{
				Message: "egress-allowed",
				Data:    lager.Data{"source": container, "packet": parsedData},
			}))
		})
	})

	Context("when the data is for a denied packet", func() {
		BeforeEach(func() {
			parsedData.Allowed = false
		})
		It("merges the data", func() {
			merged, err := logMerger.Merge(parsedData)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeContainerRepo.GetByIPCallCount()).To(Equal(1))
			Expect(fakeContainerRepo.GetByIPArgsForCall(0)).To(Equal("5.6.7.8"))

			Expect(merged).To(Equal(merger.IPTablesLogData{
				Message: "ingress-denied",
				Data:    lager.Data{"destination": container, "packet": parsedData},
			}))
		})
	})

	Context("when the container repo returns an error", func() {
		BeforeEach(func() {
			fakeContainerRepo.GetByIPReturns(repository.Container{}, errors.New("banana"))
		})
		It("returns an error", func() {
			_, err := logMerger.Merge(parsedData)
			Expect(err).To(MatchError("get container by ip: banana"))
		})
	})
})
