package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"lib/marshal"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Who Am I Handler", func() {
	var (
		request           *http.Request
		handler           *handlers.WhoAmIHandler
		resp              *httptest.ResponseRecorder
		logger            *lagertest.TestLogger
		tokenData         uaa_client.CheckTokenResponse
		fakeMetricsSender *fakes.MetricsSender
	)

	BeforeEach(func() {
		var err error
		request, err = http.NewRequest("GET", "/networking/v0/whoami", bytes.NewBuffer([]byte{}))
		Expect(err).NotTo(HaveOccurred())

		logger = lagertest.NewTestLogger("test")
		fakeMetricsSender = &fakes.MetricsSender{}
		handler = &handlers.WhoAmIHandler{
			Logger:        logger,
			Marshaler:     marshal.MarshalFunc(json.Marshal),
			MetricsSender: fakeMetricsSender,
		}
		resp = httptest.NewRecorder()
		tokenData = uaa_client.CheckTokenResponse{
			Scope:    []string{"network.admin"},
			UserName: "some_user",
		}
	})

	It("returns the username", func() {
		handler.ServeHTTP(resp, request, tokenData)

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(Equal(`{"user_name":"some_user"}`))
	})

	Context("when json marshaling the response fails", func() {
		BeforeEach(func() {
			handler.Marshaler = marshal.MarshalFunc(func(input interface{}) ([]byte, error) {
				return nil, errors.New("banana")
			})
		})

		It("returns a 500 status code", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		})

		It("logs the error", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(logger).To(gbytes.Say("marshal-response.*banana"))
		})

		It("increments the error counter", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(fakeMetricsSender.IncrementCounterCallCount()).To(Equal(1))
			Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("ExternalPoliciesWhoAmIError"))
		})
	})
})
