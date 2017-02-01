package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"policy-server/models"
	"policy-server/uaa_client"

	lfakes "lib/fakes"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Tags index handler", func() {
	var (
		allTags   []models.Tag
		request   *http.Request
		handler   *handlers.TagsIndex
		resp      *httptest.ResponseRecorder
		fakeStore *fakes.Store
		logger    *lagertest.TestLogger
		marshaler *lfakes.Marshaler
		tokenData uaa_client.CheckTokenResponse
	)

	BeforeEach(func() {
		allTags = []models.Tag{{
			ID:  "some-app-guid",
			Tag: "0001",
		}, {
			ID:  "some-other-app-guid",
			Tag: "0002",
		}}

		var err error
		request, err = http.NewRequest("GET", "/networking/v0/external/tags", nil)
		Expect(err).NotTo(HaveOccurred())

		marshaler = &lfakes.Marshaler{}
		marshaler.MarshalStub = json.Marshal

		fakeStore = &fakes.Store{}
		fakeStore.TagsReturns(allTags, nil)
		logger = lagertest.NewTestLogger("test")
		handler = &handlers.TagsIndex{
			Logger:    logger,
			Store:     fakeStore,
			Marshaler: marshaler,
		}
		resp = httptest.NewRecorder()
		tokenData = uaa_client.CheckTokenResponse{}
	})

	It("returns all the tags", func() {
		expectedResponseJSON := `{"tags": [
			{ "id": "some-app-guid", "tag": "0001" },
			{ "id": "some-other-app-guid", "tag": "0002" }
        ]}`
		handler.ServeHTTP(resp, request, tokenData)

		Expect(fakeStore.TagsCallCount()).To(Equal(1))
		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body).To(MatchJSON(expectedResponseJSON))
	})

	Context("when the store throws an error", func() {
		BeforeEach(func() {
			fakeStore.TagsReturns(nil, errors.New("banana"))
		})
		It("responds with 500", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "database read failed"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(logger).To(gbytes.Say("store-list-tags-failed.*banana"))
		})
	})

	Context("when the tags cannot be marshaled", func() {
		BeforeEach(func() {
			marshaler.MarshalStub = func(interface{}) ([]byte, error) {
				return nil, errors.New("grapes")
			}
		})

		It("responds with 500 and returns a descriptive error", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "database marshaling failed"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(logger).To(gbytes.Say("marshal-failed.*grapes"))
		})
	})
})
