package rules_test

import (
	"fmt"
	"lib/fakes"
	"lib/rules"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Locker", func() {
	var (
		locker *rules.IPTablesLocker
		flock  *fakes.FileLocker
	)
	BeforeEach(func() {
		flock = &fakes.FileLocker{}
		locker = &rules.IPTablesLocker{
			FileLocker: flock,
			Mutex:      &sync.Mutex{},
		}
	})
	Describe("Lifecycle", func() {
		It("locks and unlocks", func() {
			err := locker.Lock()
			Expect(err).NotTo(HaveOccurred())

			Expect(flock.OpenCallCount()).To(Equal(1))
		})
		Context("when fileLocker fails to open", func() {
			BeforeEach(func() {
				flock.OpenReturns(nil, fmt.Errorf("banana"))

			})
			It("should return the error", func() {
				err := locker.Lock()
				Expect(err).To(MatchError("open lock file: banana"))
			})
		})
	})
})
