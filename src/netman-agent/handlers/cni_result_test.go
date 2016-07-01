package handlers_test

import (
	"bytes"
	"lib/testsupport"
	"net/http"
	"net/http/httptest"
	"netman-agent/handlers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("CNI Result", func() {
	var (
		request *http.Request
		handler *handlers.CNIResult
		resp    *httptest.ResponseRecorder
		logger  *lagertest.TestLogger
		err     error
	)

	BeforeEach(func() {
		resp = httptest.NewRecorder()
		logger = lagertest.NewTestLogger("test")
		handler = &handlers.CNIResult{Logger: logger}
	})

	It("records the container info from cni add results", func() {
		requestJSON := `{
			"container_id": "foo",
			"group_id": "bar",
			"ip": "9.8.7.6"
		}`
		request, err = http.NewRequest("POST", "/cni_result", bytes.NewBuffer([]byte(requestJSON)))
		Expect(err).NotTo(HaveOccurred())

		handler.ServeHTTP(resp, request)

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(MatchJSON("{}"))

		Expect(logger).To(gbytes.Say(`cni_result_add.*container_id.*foo.*group_id.*bar.*ip.*9.8.7.6`))
	})

	It("records the container info from cni del results", func() {
		requestJSON := `{
			"container_id": "foo"
		}`
		request, err = http.NewRequest("DELETE", "/cni_result", bytes.NewBuffer([]byte(requestJSON)))
		Expect(err).NotTo(HaveOccurred())

		handler.ServeHTTP(resp, request)

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(MatchJSON("{}"))

		Expect(logger).To(gbytes.Say(`cni_result_del.*container_id.*foo`))
	})

	Context("when a request body cannot be deserialized from JSON", func() {
		It("responds with a useful status code", func() {
			request, err = http.NewRequest("POST", "/cni_result", bytes.NewBuffer([]byte(`{{{`)))
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Body.String()).To(MatchJSON(`{ "error": "cannot unmarshal request body as JSON" }`))
		})
	})

	Context("when reading the request body fails", func() {
		It("responds with a useful status code", func() {
			request, err = http.NewRequest("POST", "/cni_result", &testsupport.BadReader{})
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{ "error": "body read failed" }`))
			Expect(logger).To(gbytes.Say(`body-read.*banana`))
		})
	})

	Context("when a request has an unexpected method", func() {
		It("responds with a useful status code", func() {
			request, err = http.NewRequest("PUT", "/cni_result", bytes.NewBuffer([]byte(`{}`)))
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusMethodNotAllowed))
			Expect(resp.Body.String()).To(MatchJSON("{}"))
		})
	})
})
