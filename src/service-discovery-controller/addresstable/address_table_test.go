package addresstable_test

import (
	"fmt"
	"math/rand"
	"service-discovery-controller/addresstable"
	"sync"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AddressTable", func() {
	var (
		table              *addresstable.AddressTable
		fakeClock          *fakeclock.FakeClock
		stalenessThreshold time.Duration
		pruningInterval    time.Duration
		resumePruningDelay time.Duration
		logger             *lagertest.TestLogger
	)
	BeforeEach(func() {
		fakeClock = fakeclock.NewFakeClock(time.Now())
		stalenessThreshold = 5 * time.Second
		pruningInterval = 1 * time.Second
		resumePruningDelay = 30 * time.Second
		logger = lagertest.NewTestLogger("test")
		table = addresstable.NewAddressTable(stalenessThreshold, pruningInterval, resumePruningDelay, fakeClock, logger)
	})
	AfterEach(func() {
		table.Shutdown()
	})

	Describe("Warmth", func() {
		It("returns false initially", func() {
			Expect(table.IsWarm()).To(BeFalse())
		})
		Context("when SetWarm is called", func() {
			It("returns true", func() {
				table.SetWarm()
				Expect(table.IsWarm()).To(BeTrue())
			})
		})
	})

	Describe("Add", func() {
		It("adds an endpoint", func() {
			table.Add([]string{"foo.com"}, "192.0.0.1")
			Expect(table.Lookup("foo.com.")).To(Equal([]string{"192.0.0.1"}))
		})

		Context("when two hostnames are registered to same ip address", func() {
			It("returns both IPs", func() {
				table.Add([]string{"foo.com", "bar.com"}, "192.0.0.2")
				Expect(table.Lookup("foo.com.")).To(Equal([]string{"192.0.0.2"}))
				Expect(table.Lookup("bar.com.")).To(Equal([]string{"192.0.0.2"}))
			})
		})

		Context("when two different ips are registered to same host name", func() {
			It("returns both IPs", func() {
				table.Add([]string{"foo.com"}, "192.0.0.1")
				table.Add([]string{"foo.com"}, "192.0.0.2")
				Expect(table.Lookup("foo.com.")).To(Equal([]string{"192.0.0.1", "192.0.0.2"}))
			})
		})

		Context("when ip address is already registered", func() {
			It("ignores the duplicate ip", func() {
				table.Add([]string{"foo.com"}, "192.0.0.1")
				table.Add([]string{"foo.com"}, "192.0.0.1")
				Expect(table.Lookup("foo.com")).To(Equal([]string{"192.0.0.1"}))
			})
		})
	})

	Describe("GetAllAddresses", func() {
		BeforeEach(func() {
			table.Add([]string{"foo.com"}, "192.0.0.1")
			table.Add([]string{"foo.com"}, "192.0.0.2")
			table.Add([]string{"bar.com"}, "192.0.0.4")
		})

		It("returns all addresses", func() {
			Expect(table.GetAllAddresses()).To(Equal(map[string][]string{
				"foo.com.": {"192.0.0.1", "192.0.0.2"},
				"bar.com.": {"192.0.0.4"},
			}))
		})
	})

	Describe("Remove", func() {
		It("removes an endpoint", func() {
			table.Add([]string{"foo.com"}, "192.0.0.1")
			table.Remove([]string{"foo.com"}, "192.0.0.1")
			Expect(table.Lookup("foo.com")).To(Equal([]string{}))
		})
		Context("when two hostnames are registered to same ip address", func() {
			BeforeEach(func() {
				table.Add([]string{"foo.com.", "bar.com"}, "192.0.0.2")
			})
			It("removes both IPs", func() {
				table.Remove([]string{"foo.com", "bar.com."}, "192.0.0.2")

				Expect(table.Lookup("foo.com")).To(Equal([]string{}))
				Expect(table.Lookup("bar.com")).To(Equal([]string{}))
			})
		})

		Context("when removing an IP for an endpoint for a hostname that has multiple endpoints", func() {
			BeforeEach(func() {
				table.Add([]string{"foo.com"}, "192.0.0.3")
				table.Add([]string{"foo.com"}, "192.0.0.4")
			})
			It("removes only the IPs", func() {
				table.Remove([]string{"foo.com"}, "192.0.0.3")
				Expect(table.Lookup("foo.com")).To(Equal([]string{"192.0.0.4"}))
			})
		})

		Context("when removing an IP that does not exist", func() {
			BeforeEach(func() {
				table.Add([]string{"foo.com"}, "192.0.0.2")
			})
			It("does not panic", func() {
				table.Remove([]string{"foo.com"}, "192.0.0.1")
				Expect(table.Lookup("foo.com")).To(Equal([]string{"192.0.0.2"}))
			})
		})

		Context("when removing a host that does not exist", func() {
			It("does not panic", func() {
				table.Remove([]string{"foo.com"}, "192.0.0.1")
				Expect(table.Lookup("foo.com")).To(Equal([]string{}))
			})
		})
	})

	Describe("Lookup", func() {
		It("returns an empty array for an unknown hostname", func() {
			Expect(table.Lookup("foo.com")).To(Equal([]string{}))
		})
		Context("when routes go stale", func() {
			BeforeEach(func() {
				table.Add([]string{"stale.com"}, "192.0.0.1")
				table.Add([]string{"fresh.updated.com"}, "192.0.0.2")

				fakeClock.Increment(stalenessThreshold - 1*time.Second)

				By("adding/updating routes to make them fresh", func() {
					table.Add([]string{"fresh.updated.com"}, "192.0.0.2")
					table.Add([]string{"fresh.just.added.com"}, "192.0.0.3")
				})

				fakeClock.Increment(1001 * time.Millisecond)
			})
			It("prunes stale routes", func() {
				Eventually(func() []string { return table.Lookup("stale.com") }).Should(Equal([]string{}))
				Eventually(func() []string { return table.Lookup("fresh.updated.com") }).Should(Equal([]string{"192.0.0.2"}))
				Eventually(func() []string { return table.Lookup("fresh.just.added.com") }).Should(Equal([]string{"192.0.0.3"}))
			})
			It("logs pruned addresses to DEBUG", func() {
				Eventually(func() []string { return table.Lookup("stale.com") }).Should(Equal([]string{}))
				Expect(logger.Logs()).Should(Not(BeEmpty()))
				pruneMessage := logger.Logs()[0]
				Expect(pruneMessage.LogLevel).To(Equal(lager.DEBUG))
				Expect(pruneMessage.Message).To(ContainSubstring("pruning address 192.0.0.1 from stale.com"))
			})
		})
	})

	Describe("PausePruning", func() {
		BeforeEach(func() {
			table.Add([]string{"stale.com"}, "192.0.0.1")
		})
		It("does not prune stale routes", func() {
			table.PausePruning()

			fakeClock.Increment(stalenessThreshold + 1*time.Second)
			Consistently(func() []string { return table.Lookup("stale.com") }).Should(Equal([]string{"192.0.0.1"}))
		})
	})

	Describe("ResumePruning", func() {
		Context("when pruning is initially paused", func() {
			BeforeEach(func() {
				table.Add([]string{"stale.com"}, "192.0.0.1")
				table.PausePruning()
				fakeClock.Increment(stalenessThreshold + 1*time.Second)
			})
			It("starts pruning again", func() {
				table.ResumePruning()
				Consistently(func() []string { return table.Lookup("stale.com") }).Should(Equal([]string{"192.0.0.1"}))
				fakeClock.Increment(resumePruningDelay - 1*time.Second)
				Consistently(func() []string { return table.Lookup("stale.com") }).Should(Equal([]string{"192.0.0.1"}))
				fakeClock.Increment(2 * time.Second)
				Eventually(func() []string { return table.Lookup("stale.com") }).Should(Equal([]string{}))
			})
		})
	})

	Describe("Shutdown", func() {
		It("stops pruning", func() {
			table.Add([]string{"foo.com"}, "192.0.0.1")
			table.Shutdown()
			fakeClock.Increment(stalenessThreshold + time.Second)
			Consistently(func() []string { return table.Lookup("foo.com") }).Should(Equal([]string{"192.0.0.1"}))
			Expect(fakeClock.WatcherCount()).To(Equal(0))
		})
	})

	Describe("Concurrency", func() {
		It("keeps the datastore consistent", func() {
			var wg sync.WaitGroup
			wg.Add(4)
			go func() {
				table.ResumePruning()
				wg.Done()
			}()
			go func() {
				table.Add([]string{"foo.com"}, "192.0.0.2")
				wg.Done()
			}()
			go func() {
				table.Add([]string{"foo.com"}, "192.0.0.1")
				wg.Done()
			}()
			go func() {
				fakeClock.Increment(stalenessThreshold - time.Second)
				wg.Done()
			}()
			Eventually(func() []string { return table.Lookup("foo.com") }).Should(ConsistOf([]string{
				"192.0.0.1",
				"192.0.0.2",
			}))
			wg.Wait()

			wg.Add(3)
			table.Add([]string{"foo.com"}, "192.0.0.2")
			go func() {
				fakeClock.Increment(stalenessThreshold - time.Second)
				wg.Done()
			}()
			go func() {
				table.Remove([]string{"foo.com"}, "192.0.0.2")
				wg.Done()
			}()
			go func() {
				table.Remove([]string{"foo.com"}, "192.0.0.1")
				wg.Done()
			}()
			Eventually(func() []string { return table.Lookup("foo.com") }).Should(ConsistOf([]string{}))
			wg.Wait()
		})
		It("does not deadlock in the face of multiple concurrent operations", func() {
			var wg sync.WaitGroup
			const nRoutines = 30
			wg.Add(nRoutines)
			for r := 0; r < nRoutines; r++ {
				go func(i int) {
					switch rand.Intn(6) {
					case 0:
						table.Add([]string{fmt.Sprintf("%d-foo.com", i)}, fmt.Sprintf("192.0.0.%d", i))
					case 1:
						table.Remove([]string{fmt.Sprintf("%d-foo.com", i)}, fmt.Sprintf("192.0.0.%d", i))
					case 2:
						fakeClock.Increment(stalenessThreshold / 2)
					case 3:
						table.PausePruning()
					case 4:
						table.ResumePruning()
					case 5:
						table.GetAllAddresses()
					}
					wg.Done()
				}(r)
			}
			wg.Wait()
		})
	})

	Describe("Warm Concurrency", func() {
		It("does not deadlock in the face of multiple concurrent operations", func() {
			var wg sync.WaitGroup
			const nRoutines = 10
			wg.Add(nRoutines)
			for r := 0; r < nRoutines; r++ {
				go func(i int) {
					if i%2 == 0 {
						table.SetWarm()
					} else {
						table.IsWarm()
					}
					wg.Done()
				}(r)
			}
			wg.Wait()
			Expect(table.IsWarm()).To(BeTrue())
		})
	})
})
