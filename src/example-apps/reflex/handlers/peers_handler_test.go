package handlers_test

import (
	"errors"
	"example-apps/reflex/fakes"
	"example-apps/reflex/handlers"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("PeersHandler", func() {
	var (
		store   *fakes.Store
		handler *handlers.PeersHandler
		resp    *httptest.ResponseRecorder
		req     *http.Request
		logger  *lagertest.TestLogger
	)

	BeforeEach(func() {
		store = &fakes.Store{}
		store.GetAddressesReturns([]string{"1.2.3.4", "5.6.7.8"})

		logger = lagertest.NewTestLogger("test")

		handler = &handlers.PeersHandler{Logger: logger, Store: store}
		resp = httptest.NewRecorder()

		var err error
		req, err = http.NewRequest("GET", "/peers", nil)
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns the list of addresses from the store", func() {
		handler.ServeHTTP(resp, req)

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(MatchJSON(`{"ips" : ["1.2.3.4", "5.6.7.8"]}`))
	})

	Context("when writing the response body fails", func() {
		var resp *fakes.ResponseWriter

		BeforeEach(func() {
			resp = &fakes.ResponseWriter{}
			resp.WriteReturns(0, errors.New("banana"))
		})

		It("logs the error", func() {
			handler.ServeHTTP(resp, req)
			Expect(logger).To(gbytes.Say("error.*banana"))
		})
	})
})
