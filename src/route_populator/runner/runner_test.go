package runner_test

import (
	"errors"
	"route_populator/publisher"
	"route_populator/publisher/fakes"
	"route_populator/runner"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Runner", func() {
	numGoRoutines := 2

	validJob := publisher.Job{
		PublishingEndpoint: "pub.end.point",

		BackendHost: "1.2.3.4",
		BackendPort: 1234,

		AppDomain:  "apps.com",
		AppName:    "some-app",
		StartRange: 500,
		EndRange:   505,
	}
	publishDelay := 0 * time.Second

	Describe("Start", func() {
		It("publishes once as soon as the runner is started", func(done Done) {
			defer close(done)
			var allMessages string
			msgLock := &sync.Mutex{}
			c := &fakes.FakePublishingConnection{}
			c.PublishStub = func(subj string, data []byte) error {
				msgLock.Lock()
				allMessages = allMessages + string(data)
				msgLock.Unlock()
				return nil
			}
			createConnection := func(endpoint string) (publisher.PublishingConnection, error) {
				return c, nil
			}

			r := runner.NewRunner(createConnection, validJob, numGoRoutines, 10*time.Second, publishDelay)
			err := r.Start()
			Expect(err).ToNot(HaveOccurred())
			r.Stop()
			err = r.Wait()
			Expect(err).ToNot(HaveOccurred())
			Expect(c.PublishCallCount()).To(Equal(5))
			for i := validJob.StartRange; i < validJob.EndRange; i += 1 {
				Expect(allMessages).To(ContainSubstring("some-app-%d.apps.com", i))
			}
		}, 1)
		It("publishes on an interval", func(done Done) {
			defer close(done)
			c := &fakes.FakePublishingConnection{}
			createConnection := func(endpoint string) (publisher.PublishingConnection, error) {
				return c, nil
			}

			r := runner.NewRunner(createConnection, validJob, numGoRoutines, 600*time.Millisecond, publishDelay)
			err := r.Start()
			Expect(err).ToNot(HaveOccurred())
			time.Sleep(time.Second)
			r.Stop()
			err = r.Wait()
			Expect(err).ToNot(HaveOccurred())
			Expect(c.PublishCallCount()).Should(Equal(10))
		}, 2)
	})
	Describe("Wait", func() {
		It("returns an error if initializing fails", func(done Done) {
			defer close(done)
			createConnection := func(endpoint string) (publisher.PublishingConnection, error) {
				return nil, errors.New("Some failure")
			}

			r := runner.NewRunner(createConnection, validJob, numGoRoutines, 600*time.Millisecond, publishDelay)
			err := r.Start()
			Expect(err).ToNot(HaveOccurred())
			r.Stop()
			err = r.Wait()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("initializing connection: Some failure"))
		}, 2)

		It("returns an error if publishing fails", func(done Done) {
			defer close(done)
			c := &fakes.FakePublishingConnection{}
			c.PublishStub = func(subj string, data []byte) error {
				return errors.New("Some failure")
			}
			createConnection := func(endpoint string) (publisher.PublishingConnection, error) {
				return c, nil
			}

			r := runner.NewRunner(createConnection, validJob, numGoRoutines, 600*time.Millisecond, publishDelay)
			err := r.Start()
			Expect(err).ToNot(HaveOccurred())
			r.Stop()
			err = r.Wait()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("publishing: Some failure"))
		}, 2)
	})
	Describe("Stop", func() {
		It("terminates the runner", func(done Done) {
			defer close(done)
			createConnection := func(endpoint string) (publisher.PublishingConnection, error) {
				return &fakes.FakePublishingConnection{}, nil
			}

			r := runner.NewRunner(createConnection, validJob, numGoRoutines, 1*time.Second, publishDelay)
			err := r.Start()
			Expect(err).ToNot(HaveOccurred())
			r.Stop()
			err = r.Wait()
			Expect(err).ToNot(HaveOccurred())
		}, 1)
		It("prevents the runner from being started again", func(done Done) {
			defer close(done)

			createConnection := func(endpoint string) (publisher.PublishingConnection, error) {
				return &fakes.FakePublishingConnection{}, nil
			}

			r := runner.NewRunner(createConnection, validJob, numGoRoutines, 1*time.Second, publishDelay)
			err := r.Start()
			Expect(err).ToNot(HaveOccurred())
			r.Stop()
			err = r.Wait()
			Expect(err).ToNot(HaveOccurred())

			err = r.Start()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("Cannot restart a runner."))
		}, 1)
	})
	Describe("PartitionRange", func() {
		It("returns a single range", func() {
			r := runner.PartitionRange(0, 20, 20)
			Expect(r).To(Equal([]int{0, 20}))
		})
		It("return multiple start indexes, and the last index is an end index", func() {
			r := runner.PartitionRange(0, 20, 10)
			Expect(r).To(Equal([]int{0, 10, 20}))
		})
		It("returns a full range if the partition size does not divide the range evenly", func() {
			r := runner.PartitionRange(0, 27, 13)
			Expect(r).To(Equal([]int{0, 13, 27}))
		})
	})
})
