package db_test

import (
	"encoding/json"
	"lib/db"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	var config db.Config
	var expectedJSON string

	BeforeEach(func() {
		config = db.Config{
			Host:     "example.com",
			Port:     9953,
			Username: "bob",
			Password: "secret",
			Name:     "database1",
			SSLMode:  "false",
		}

		expectedJSON = `{
			"host": "example.com",
			"port": 9953,
			"username": "bob",
			"password": "secret",
			"name": "database1",
			"ssl_mode": "false"
		}`
	})

	It("serializes and deserializes", func() {
		bytes, err := json.Marshal(config)
		Expect(err).NotTo(HaveOccurred())
		Expect(bytes).To(MatchJSON(expectedJSON))

		var config2 db.Config
		err = json.Unmarshal([]byte(expectedJSON), &config2)
		Expect(err).NotTo(HaveOccurred())
		Expect(config).To(Equal(config2))
	})

	It("generates a Postgres URL", func() {
		url, err := config.PostgresURL()
		Expect(err).NotTo(HaveOccurred())

		Expect(url).To(Equal(`postgres://bob:secret@example.com:9953/database1?sslmode=false`))
	})

	DescribeTable("missing or invalid config",
		func(expectedError string, corrupter func()) {
			corrupter()
			_, err := config.PostgresURL()
			Expect(err).To(MatchError(expectedError))
		},

		Entry("missing Host", `"host" is required`, func() { config.Host = "" }),
		Entry("missing Port", `"port" is required`, func() { config.Port = 0 }),
		Entry("missing Username", `"username" is required`, func() { config.Username = "" }),
		Entry("missing Name", `"name" is required`, func() { config.Name = "" }),
		Entry("missing SSLMode", `"ssl_mode" is required`, func() { config.SSLMode = "" }),
	)
})
