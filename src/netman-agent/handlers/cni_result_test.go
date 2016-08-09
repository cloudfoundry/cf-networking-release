package handlers_test

import (
	"bytes"
	"errors"
	"lib/testsupport"
	"net/http"
	"net/http/httptest"
	"netman-agent/fakes"
	"netman-agent/handlers"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("CNI Result", func() {
	var (
		request     *http.Request
		handler     *handlers.CNIResult
		resp        *httptest.ResponseRecorder
		logger      *lagertest.TestLogger
		storeWriter *fakes.StoreWriter
		err         error
	)

	BeforeEach(func() {
		resp = httptest.NewRecorder()
		logger = lagertest.NewTestLogger("test")
		storeWriter = &fakes.StoreWriter{}
		handler = &handlers.CNIResult{
			Logger:      logger,
			StoreWriter: storeWriter,
		}
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

	It("updates the store on CNI ADD", func() {
		requestJSON := `{
			"container_id": "foo",
			"group_id": "bar",
			"ip": "9.8.7.6"
		}`
		request, err = http.NewRequest("POST", "/cni_result", bytes.NewBuffer([]byte(requestJSON)))
		Expect(err).NotTo(HaveOccurred())

		handler.ServeHTTP(resp, request)

		Expect(storeWriter.AddCallCount()).To(Equal(1))
		containerID, groupID, IP := storeWriter.AddArgsForCall(0)
		Expect(containerID).To(Equal("foo"))
		Expect(groupID).To(Equal("bar"))
		Expect(IP).To(Equal("9.8.7.6"))
	})

	It("updates the store on CNI DEL", func() {
		requestJSON := `{
			"container_id": "foo"
		}`
		request, err = http.NewRequest("DELETE", "/cni_result", bytes.NewBuffer([]byte(requestJSON)))
		Expect(err).NotTo(HaveOccurred())

		handler.ServeHTTP(resp, request)

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(MatchJSON("{}"))

		Expect(logger).To(gbytes.Say(`cni_result_del.*container_id.*foo`))

		Expect(storeWriter.DelCallCount()).To(Equal(1))
		containerID := storeWriter.DelArgsForCall(0)
		Expect(containerID).To(Equal("foo"))
	})

	Context("when adding to the store fails", func() {
		BeforeEach(func() {
			storeWriter.AddReturns(errors.New("potato"))
		})
		It("responds with a 500 status code and logs the error", func() {
			requestJSON := `{
			"container_id": "foo",
			"group_id": "bar",
			"ip": "9.8.7.6"
		}`
			request, err = http.NewRequest("POST", "/cni_result", bytes.NewBuffer([]byte(requestJSON)))
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(logger).To(gbytes.Say(`store-add.*potato`))
		})
	})

	Context("when deleting from the store fails", func() {
		BeforeEach(func() {
			storeWriter.DelReturns(errors.New("potato"))
		})
		It("responds with a 500 status code and logs the error", func() {
			requestJSON := `{
			"container_id": "foo"
		}`
			request, err = http.NewRequest("DELETE", "/cni_result", bytes.NewBuffer([]byte(requestJSON)))
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(logger).To(gbytes.Say(`store-del.*potato`))
		})
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
