package port_allocator_test

import (
	"errors"
	"garden-external-networker/fakes"
	"garden-external-networker/port_allocator"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PortAllocator", func() {
	var (
		portAllocator *port_allocator.PortAllocator
		tracker       *fakes.Tracker
		serializer    *fakes.Serializer
		locker        *fakes.FileLocker
		lockedFile    *os.File
	)
	BeforeEach(func() {
		serializer = &fakes.Serializer{}
		tracker = &fakes.Tracker{}
		locker = &fakes.FileLocker{}
		serializer.DecodeAllReturns(nil)
		tracker.AcquireOneReturns(111, nil)

		portAllocator = &port_allocator.PortAllocator{
			Tracker:    tracker,
			Serializer: serializer,
			Locker:     locker,
		}

		lockedFile = &os.File{}
		locker.OpenReturns(lockedFile, nil)
	})

	Describe("AllocatePort", func() {
		It("deserializes the pool from the locked file", func() {
			_, err := portAllocator.AllocatePort("some-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(serializer.DecodeAllCallCount()).To(Equal(1))

			file, _ := serializer.DecodeAllArgsForCall(0)
			Expect(file).To(Equal(lockedFile))
		})

		It("acquires the port from the pool", func() {
			_, err := portAllocator.AllocatePort("some-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(serializer.DecodeAllCallCount()).To(Equal(1))
			Expect(tracker.AcquireOneCallCount()).To(Equal(1))

			_, pool := serializer.DecodeAllArgsForCall(0)
			Expect(tracker.AcquireOneArgsForCall(0)).To(Equal(pool))
		})

		It("re-serializes the pool to the locked file", func() {
			_, err := portAllocator.AllocatePort("some-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(serializer.EncodeAndOverwriteCallCount()).To(Equal(1))

			_, poolForDecode := serializer.DecodeAllArgsForCall(0)
			file, poolForEncode := serializer.EncodeAndOverwriteArgsForCall(0)

			Expect(file).To(Equal(lockedFile))
			Expect(poolForEncode).To(Equal(poolForDecode))
		})

		It("returns the port", func() {
			port, err := portAllocator.AllocatePort("some-handle")
			Expect(err).NotTo(HaveOccurred())
			Expect(port).To(Equal(111))
		})

		It("closes (and thus unlocks) the file", func() {
			file, err := ioutil.TempFile("", "")
			Expect(err).NotTo(HaveOccurred())

			locker.OpenReturns(file, nil)
			_, err = portAllocator.AllocatePort("some-handle")
			Expect(err).NotTo(HaveOccurred())

			By("checking that the write to the closed file should fail")
			_, err = file.WriteString("foo")
			Expect(err).To(HaveOccurred())
		})

		Context("when the locker fails to open the file", func() {
			BeforeEach(func() {
				locker.OpenReturns(nil, errors.New("potato"))
			})
			It("wraps and returns the error", func() {
				_, err := portAllocator.AllocatePort("some-handle")
				Expect(err).To(MatchError("open lock: potato"))
			})
		})

		Context("when the serializer fails to decode", func() {
			BeforeEach(func() {
				serializer.DecodeAllReturns(errors.New("potato"))
			})
			It("wraps and returns the error", func() {
				_, err := portAllocator.AllocatePort("some-handle")
				Expect(err).To(MatchError("decoding state file: potato"))
			})
		})

		Context("when the tracker cannot acquire a port", func() {
			BeforeEach(func() {
				tracker.AcquireOneReturns(0, errors.New("turnip"))
			})
			It("wraps and returns the error", func() {
				_, err := portAllocator.AllocatePort("some-handle")
				Expect(err).To(MatchError("acquire port: turnip"))
			})
		})

		Context("when serializing the pool fails", func() {
			BeforeEach(func() {
				serializer.EncodeAndOverwriteReturns(errors.New("turnip"))
			})
			It("wraps and returns the error", func() {
				_, err := portAllocator.AllocatePort("some-handle")
				Expect(err).To(MatchError("encode and overwrite: turnip"))
			})
		})
	})
})
