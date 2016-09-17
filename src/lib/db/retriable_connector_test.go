package db_test

import (
	"errors"
	"lib/db"
	"lib/fakes"
	"time"

	"github.com/jmoiron/sqlx"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("RetriableConnector", func() {
	var (
		sleeper            *fakes.Sleeper
		retriableConnector *db.RetriableConnector
		numTries           int
	)

	BeforeEach(func() {
		sleeper = &fakes.Sleeper{}
		numTries = 0

		retriableConnector = &db.RetriableConnector{
			Sleeper:       sleeper,
			RetryInterval: time.Minute,
			Connector: func(db.Config) (*sqlx.DB, error) {
				numTries++
				if numTries > 3 {
					return nil, nil
				}
				return nil, db.RetriableError{Inner: errors.New("welp")}
			},
		}
	})

	Context("when the inner Connector returns a non-retriable error", func() {
		It("returns the error immediately", func() {
			retriableConnector := db.RetriableConnector{
				Connector: func(db.Config) (*sqlx.DB, error) {
					return nil, errors.New("banana")
				},
			}

			_, err := retriableConnector.GetConnectionPool(db.Config{ConnectionString: "whatever"})
			Expect(err).To(MatchError("banana"))
		})
	})

	Context("when the inner Connector returns a retriable error", func() {
		It("retries the connection", func() {
			retriableConnector.MaxRetries = 5

			_, err := retriableConnector.GetConnectionPool(db.Config{ConnectionString: "whatever"})

			Expect(numTries).To(Equal(4))
			Expect(err).NotTo(HaveOccurred())
		})

		It("waits between retries", func() {
			retriableConnector.MaxRetries = 5

			_, err := retriableConnector.GetConnectionPool(db.Config{ConnectionString: "whatever"})
			Expect(err).To(Succeed())

			Expect(sleeper.SleepCallCount()).To(Equal(3))
			Expect(sleeper.SleepArgsForCall(0)).To(Equal(time.Minute))
			Expect(sleeper.SleepArgsForCall(1)).To(Equal(time.Minute))
			Expect(sleeper.SleepArgsForCall(2)).To(Equal(time.Minute))
		})

		Context("when max retries have occurred", func() {
			It("stops retrying and returns the last error", func() {
				retriableConnector.MaxRetries = 10
				retriableConnector.Connector = func(db.Config) (*sqlx.DB, error) {
					numTries++
					return nil, db.RetriableError{Inner: errors.New("welp")}
				}

				_, err := retriableConnector.GetConnectionPool(db.Config{ConnectionString: "whatever"})
				Expect(err).To(MatchError(db.RetriableError{Inner: errors.New("welp")}))

				Eventually(numTries).Should(Equal(10))
				Expect(sleeper.SleepCallCount()).To(Equal(9))
			})
		})
	})
})
