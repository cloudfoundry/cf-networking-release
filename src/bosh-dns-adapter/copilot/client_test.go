package copilot_test

import (
	. "bosh-dns-adapter/copilot"
	"bosh-dns-adapter/copilot/api"
	"bosh-dns-adapter/copilot/fakes"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Copilot Client", func() {
	Context("GetVIPByName", func() {
		var (
			copilotClient                *Client
			fakeVIPResolverCopilotClient *fakes.VIPResolverCopilotClient
			response                     *api.GetVIPByNameResponse
			responseErr                  error
		)

		BeforeEach(func() {
			fakeVIPResolverCopilotClient = &fakes.VIPResolverCopilotClient{}
			copilotClient = &Client{
				VIPResolverCopilotClient: fakeVIPResolverCopilotClient,
			}
			response = &api.GetVIPByNameResponse{Ip: "192.168.1.1"}
			responseErr = errors.New("response error")
			fakeVIPResolverCopilotClient.GetVIPByNameReturns(response, responseErr)
		})

		It("constructs a request from the args and passes to the grpc client", func() {
			copilotClient.IP("apps.istio.local.")
			_, req, _ := fakeVIPResolverCopilotClient.GetVIPByNameArgsForCall(0)
			Expect(req.GetFqdn()).To(Equal("apps.istio.local."))
		})

		It("returns and ip from the server", func() {
			ip, err := copilotClient.IP("app.istio.local.")
			Expect(ip).To(Equal("192.168.1.1"))
			Expect(err).To(Equal(responseErr))
		})
	})

	Context("NewConnectedClient", func() {
		Context("when building a connection succeeds", func() {
			It("constructs a client with a TLS dialed connection", func() {
				server := ghttp.NewTLSServer()
				defer server.Close()

				client, err := NewConnectedClient(server.Addr(), WithTLSConfig(server.HTTPTestServer.TLS))
				Expect(err).NotTo(HaveOccurred())
				Expect(client).NotTo(BeNil())
			})

			It("fails to construct a client with non-TLS dialed connection", func() {
				server := ghttp.NewServer()
				defer server.Close()

				_, err := NewConnectedClient(server.Addr())
				Expect(err).To(MatchError(ContainSubstring("no transport security set")))
			})
		})
	})

	Context("Close", func() {
		It("closes the grpc connection", func() {
			server := ghttp.NewTLSServer()

			client, err := NewConnectedClient(server.Addr(), WithTLSConfig(server.HTTPTestServer.TLS))
			Expect(err).NotTo(HaveOccurred())

			err = client.Close()
			Expect(err).ToNot(HaveOccurred())

			err = client.Close()
			Expect(err).To(MatchError(ContainSubstring("grpc: the client connection is closing")))
		})
	})
})
