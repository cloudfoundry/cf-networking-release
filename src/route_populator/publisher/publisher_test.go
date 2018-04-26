package publisher_test

import (
	"errors"
	"fmt"
	"route_populator/publisher"
	"route_populator/publisher/fakes"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Worker", func() {
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
	Describe("Initialize", func() {
		It("errors if validation of the job properties fails", func() {
			w := publisher.NewPublisher(publisher.Job{
				PublishingEndpoint: "endpoint",
				BackendHost:        "1.2.3.4",
				BackendPort:        1234,
			}, publishDelay)
			createConnection := func(endpoint string) (publisher.PublishingConnection, error) {
				return nil, nil
			}
			err := w.Initialize(createConnection)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(`Invalid job properties: Missing "AppDomain"`))
		})

		It("errors if the creation of a connection fails", func() {
			w := publisher.NewPublisher(validJob, publishDelay)
			createConnection := func(endpoint string) (publisher.PublishingConnection, error) {
				return nil, errors.New("Unable to create connection")
			}
			err := w.Initialize(createConnection)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("Unable to create connection"))
		})
	})

	Describe("PublishRouteRegistrations", func() {
		It("correctly publishes (endrange - startrange) register messages", func() {
			w := publisher.NewPublisher(validJob, publishDelay)
			c := &fakes.FakePublishingConnection{}
			createConnection := func(endpoint string) (publisher.PublishingConnection, error) {
				Expect(endpoint).To(Equal("pub.end.point"))
				return c, nil
			}
			err := w.Initialize(createConnection)
			Expect(err).ToNot(HaveOccurred())

			err = w.PublishRouteRegistrations()
			Expect(err).ToNot(HaveOccurred())

			Expect(c.PublishCallCount()).To(Equal(5))
			for i := 0; i < 5; i += 1 {
				msg, data := c.PublishArgsForCall(i)
				Expect(msg).To(Equal("router.register"))
				Expect(data).To(BeEquivalentTo(fmt.Sprintf("{\"host\":\"1.2.3.4\",\"port\":1234,\"uris\":[\"some-app-%d.apps.com\"]}", 500+i)))
			}
		})

		It("immediately errors if publishing fails", func() {
			w := publisher.NewPublisher(validJob, publishDelay)
			c := &fakes.FakePublishingConnection{}
			c.PublishReturns(errors.New("Unable to publish message"))

			createConnection := func(endpoint string) (publisher.PublishingConnection, error) {
				Expect(endpoint).To(Equal("pub.end.point"))
				return c, nil
			}
			err := w.Initialize(createConnection)
			Expect(err).ToNot(HaveOccurred())

			err = w.PublishRouteRegistrations()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("Unable to publish message"))

			Expect(c.PublishCallCount()).To(Equal(1))
		})
	})

	Describe("Finish", func() {
		It("closes the connection with the publishing endpoint", func() {
			w := publisher.NewPublisher(validJob, publishDelay)
			c := &fakes.FakePublishingConnection{}
			createConnection := func(endpoint string) (publisher.PublishingConnection, error) {
				Expect(endpoint).To(Equal("pub.end.point"))
				return c, nil
			}
			err := w.Initialize(createConnection)
			Expect(err).ToNot(HaveOccurred())

			w.Finish()
			Expect(c.CloseCallCount()).To(Equal(1))
		})
	})
})
