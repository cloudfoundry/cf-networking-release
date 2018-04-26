package mbus_test

import (
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"
	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/nats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "service-discovery-controller/mbus"
	"time"
)

var _ = Describe("NatsConnProvider", func() {
	var (
		provider    NatsConnProvider
		gnatsServer *server.Server
		natsCon     *nats.Conn
		port        int
	)

	BeforeEach(func() {
		port = ports.PickAPort()
		gnatsServer = RunServerOnPort(port)
		gnatsServer.Start()

		natsUrl := "nats://username:password@" + gnatsServer.Addr().String()

		provider = &NatsConnWithUrlProvider{
			Url: natsUrl,
		}
	})

	AfterEach(func() {
		if natsCon != nil {
			natsCon.Close()
		}
		gnatsServer.Shutdown()
	})

	It("returns a configured nats connection", func() {
		timeoutOption := nats.Timeout(42 * time.Second)
		conn, err := provider.Connection(timeoutOption)
		Expect(err).NotTo(HaveOccurred())
		var successfulCast bool
		natsCon, successfulCast = conn.(*nats.Conn)
		Expect(successfulCast).To(BeTrue())

		Expect(natsCon.Opts.Timeout).To(Equal(42 * time.Second))
	})
})
