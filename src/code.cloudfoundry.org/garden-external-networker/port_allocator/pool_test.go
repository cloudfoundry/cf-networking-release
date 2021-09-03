package port_allocator_test

import (
	"encoding/json"

	"code.cloudfoundry.org/garden-external-networker/port_allocator"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Tracker", func() {
	var (
		pool    *port_allocator.Pool
		tracker *port_allocator.Tracker
	)
	BeforeEach(func() {
		pool = &port_allocator.Pool{}
		tracker = &port_allocator.Tracker{
			StartPort: 100,
			Capacity:  10,
		}
	})

	Describe("AcquireOne", func() {
		It("reserves and returns a port from the pool", func() {
			newPort, err := tracker.AcquireOne(pool, "some-handle")
			Expect(err).NotTo(HaveOccurred())
			Expect(newPort).To(BeInRange(100, 110))
			Expect(pool.AcquiredPorts).To(Equal(map[int]string{newPort: "some-handle"}))
		})

		Context("when acquiring multiple ports", func() {
			It("gives unique ports", func() {
				firstPort, err := tracker.AcquireOne(pool, "some-handle")
				Expect(err).NotTo(HaveOccurred())
				secondPort, err := tracker.AcquireOne(pool, "some-handle")
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
				pool.AcquiredPorts = map[int]string{
					100: "some-handle",
					102: "some-handle",
				}
			})

			It("reserves and returns that unacquired port", func() {
				port, err := tracker.AcquireOne(pool, "some-handle")
				Expect(err).NotTo(HaveOccurred())
				Expect(port).To(Equal(101))
				Expect(pool.AcquiredPorts).To(HaveKey(101))
			})
		})

		Context("when the pool has reached capacity", func() {
			BeforeEach(func() {
				tracker.Capacity = 2
				pool.AcquiredPorts = map[int]string{
					100: "some-handle",
					101: "some-handle",
				}
			})

			It("returns a useful error", func() {
				_, err := tracker.AcquireOne(pool, "some-handle")
				Expect(err).To(Equal(port_allocator.ErrorPortPoolExhausted))
			})
		})

		Describe("performance", func() {
			Measure("should acquire all of the ports quickly", func(b Benchmarker) {
				tracker.Capacity = 4000
				runtime := b.Time("runtime", func() {
					for i := 0; i < 4000; i++ {
						_, err := tracker.AcquireOne(pool, "some-handle")
						Expect(err).NotTo(HaveOccurred())
					}
				})

				Expect(runtime.Seconds()).To(BeNumerically("<", 5), "Acquiring a port shouldn't take too long.")
			}, 10)
		})
	})

	Describe("acquire and release lifecycle", func() {
		It("can re-acquire ports which have been acquired and then released", func() {
			var err error
			for i := 0; i < tracker.Capacity; i++ {
				if i%2 == 0 {
					_, err = tracker.AcquireOne(pool, "some-handle")
				} else {
					_, err = tracker.AcquireOne(pool, "some-handle2")
				}
			}
			Expect(err).NotTo(HaveOccurred())
			Expect(tracker.ReleaseAll(pool, "some-handle")).To(Succeed())
			reacquired, err := tracker.AcquireOne(pool, "some-handle")
			Expect(err).NotTo(HaveOccurred())
			Expect(reacquired).To(Equal(100))
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
			pool.AcquiredPorts = map[int]string{
				42:  "some-handle",
				105: "some-handle2",
			}

			bytes, err := json.Marshal(pool)
			Expect(err).NotTo(HaveOccurred())

			var newPool port_allocator.Pool
			Expect(json.Unmarshal(bytes, &newPool)).To(Succeed())

			Expect(newPool.AcquiredPorts).To(Equal(pool.AcquiredPorts))
		})

		It("marshals as a map from container handle to list of allocated ports", func() {
			pool.AcquiredPorts = map[int]string{
				42:  "some-handle",
				105: "some-handle2",
			}

			bytes, err := json.Marshal(pool)
			Expect(err).NotTo(HaveOccurred())

			Expect(bytes).To(MatchJSON(`{ "acquired_ports": {
				"some-handle": [ 42 ],
				"some-handle2": [ 105 ]
			} }`))
		})
	})
})

func BeInRange(min, max int) types.GomegaMatcher {
	return SatisfyAll(
		BeNumerically(">=", min),
		BeNumerically("<", max))
}
