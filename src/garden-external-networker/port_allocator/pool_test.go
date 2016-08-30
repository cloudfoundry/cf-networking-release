package port_allocator_test

import (
	"encoding/json"
	"garden-external-networker/port_allocator"

	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Tracker", func() {
	var (
		pool    *port_allocator.Pool
		tracker *port_allocator.Tracker
		logger  *lagertest.TestLogger
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		pool = &port_allocator.Pool{}
		tracker = &port_allocator.Tracker{
			Logger:    logger,
			StartPort: 100,
			Capacity:  10,
		}
	})

	Describe("AcquireOne", func() {
		It("reserves and returns a port from the pool", func() {
			newPort, err := tracker.AcquireOne(pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(newPort).To(BeInRange(100, 110))
			Expect(pool.AcquiredPorts).To(Equal(map[int]bool{newPort: true}))
		})

		Context("when acquiring multiple ports", func() {
			It("gives unique ports", func() {
				firstPort, err := tracker.AcquireOne(pool)
				Expect(err).NotTo(HaveOccurred())
				secondPort, err := tracker.AcquireOne(pool)
				Expect(err).NotTo(HaveOccurred())

				Expect(pool.AcquiredPorts).To(HaveLen(2))
				Expect(firstPort).NotTo(Equal(secondPort))
				Expect(pool.AcquiredPorts).To(HaveKey(firstPort))
				Expect(pool.AcquiredPorts).To(HaveKey(secondPort))
			})
		})

		Context("when the only unacquired port is in the middle of the range", func() {
			BeforeEach(func() {
				tracker.Capacity = 3
				pool.AcquiredPorts = map[int]bool{
					100: true,
					102: true,
				}
			})

			It("reserves and returns that unacquired port", func() {
				port, err := tracker.AcquireOne(pool)
				Expect(err).NotTo(HaveOccurred())
				Expect(port).To(Equal(101))
				Expect(pool.AcquiredPorts).To(HaveKey(101))
			})
		})

		Context("when the pool has reached capacity", func() {
			BeforeEach(func() {
				tracker.Capacity = 2
				pool.AcquiredPorts = map[int]bool{
					100: true,
					101: true,
				}
			})

			It("returns a useful error", func() {
				_, err := tracker.AcquireOne(pool)
				Expect(err).To(Equal(port_allocator.ErrorPortPoolExhausted))
			})
		})

		Describe("performance", func() {
			Measure("should acquire all of the ports quickly", func(b Benchmarker) {
				tracker.Capacity = 4000
				runtime := b.Time("runtime", func() {
					for i := 0; i < 4000; i++ {
						_, err := tracker.AcquireOne(pool)
						Expect(err).NotTo(HaveOccurred())
					}
				})

				Expect(runtime.Seconds()).To(BeNumerically("<", 5), "Acquiring a port shouldn't take too long.")
			}, 10)
		})
	})

	Describe("acquire and release lifecycle", func() {
		It("can re-acquire ports which have been acquired and then released", func() {
			for i := 0; i < tracker.Capacity; i++ {
				_, err := tracker.AcquireOne(pool)
				Expect(err).NotTo(HaveOccurred())
			}

			Expect(tracker.ReleaseMany(pool, []int{102, 106})).To(Succeed())

			reacquired, err := tracker.AcquireOne(pool)
			Expect(err).NotTo(HaveOccurred())

			Expect(reacquired).To(SatisfyAny(
				Equal(102),
				Equal(106),
			))
		})
	})

	Context("when you try to release a port that is not acquired", func() {
		BeforeEach(func() {
			pool.AcquiredPorts = map[int]bool{105: true}
		})

		It("logs the event and succeeds", func() {
			err := tracker.ReleaseMany(pool, []int{101})
			Expect(err).NotTo(HaveOccurred())
			Expect(logger).To(gbytes.Say(`release-many.*port 101 was not previously acquired`))
		})

		It("continues releasing the other ports", func() {
			err := tracker.ReleaseMany(pool, []int{101, 105})
			Expect(err).NotTo(HaveOccurred())

			Expect(pool.AcquiredPorts).NotTo(HaveKey(105))
		})
	})

	Context("when you try to release a port outside the tracker's range", func() {
		BeforeEach(func() {
			pool.AcquiredPorts = map[int]bool{
				42:  true,
				105: true,
			}
		})
		It("logs the event and succeeds", func() {
			err := tracker.ReleaseMany(pool, []int{42})
			Expect(err).NotTo(HaveOccurred())
			Expect(logger).To(gbytes.Say(`release-many.*releasing port out of range`))
		})

		It("continues releasing the other ports", func() {
			err := tracker.ReleaseMany(pool, []int{42, 105})
			Expect(err).NotTo(HaveOccurred())

			Expect(pool.AcquiredPorts).NotTo(HaveKey(105))
		})
	})

	Describe("InRange", func() {
		It("returns true if the given port is in the allocation range", func() {
			for i := 100; i < 110; i++ {
				Expect(tracker.InRange(i)).To(BeTrue())
			}
		})
		It("otherwise returns false", func() {
			Expect(tracker.InRange(110)).To(BeFalse())
		})
	})

	Describe("serializing the pool", func() {
		It("can be roud-tripped through JSON intact", func() {
			pool.AcquiredPorts = map[int]bool{
				42:  true,
				105: true,
			}

			bytes, err := json.Marshal(pool)
			Expect(err).NotTo(HaveOccurred())

			var newPool port_allocator.Pool
			Expect(json.Unmarshal(bytes, &newPool)).To(Succeed())

			Expect(newPool.AcquiredPorts).To(Equal(pool.AcquiredPorts))
		})

		It("marshals as a list of integers", func() {
			pool.AcquiredPorts = map[int]bool{
				42:  true,
				105: true,
			}

			bytes, err := json.Marshal(pool)
			Expect(err).NotTo(HaveOccurred())

			Expect(bytes).To(MatchJSON(`{ "acquired_ports": [ 42, 105 ] }`))
		})
	})
})

func BeInRange(min, max int) types.GomegaMatcher {
	return SatisfyAll(
		BeNumerically(">=", min),
		BeNumerically("<", max))
}
