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

	hfakes "code.cloudfoundry.org/go-db-helpers/fakes"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tags index handler", func() {
	var (
		allTags           []models.Tag
		request           *http.Request
		handler           *handlers.TagsIndex
		resp              *httptest.ResponseRecorder
		fakeStore         *fakes.Store
		fakeErrorResponse *fakes.ErrorResponse
		logger            *lagertest.TestLogger
		marshaler         *hfakes.Marshaler
		tokenData         uaa_client.CheckTokenResponse
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

		marshaler = &hfakes.Marshaler{}
		marshaler.MarshalStub = json.Marshal

		fakeStore = &fakes.Store{}
		fakeErrorResponse = &fakes.ErrorResponse{}
		fakeStore.TagsReturns(allTags, nil)
		logger = lagertest.NewTestLogger("test")
		handler = &handlers.TagsIndex{
			Logger:        logger,
			Store:         fakeStore,
			Marshaler:     marshaler,
			ErrorResponse: fakeErrorResponse,
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

		It("calls the internal server error handler", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("tags-index"))
			Expect(description).To(Equal("database read failed"))
		})
	})

	Context("when the tags cannot be marshaled", func() {
		BeforeEach(func() {
			marshaler.MarshalStub = func(interface{}) ([]byte, error) {
				return nil, errors.New("grapes")
			}
		})

		It("calls the internal server error handler", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("grapes"))
			Expect(message).To(Equal("tags-index"))
			Expect(description).To(Equal("database marshaling failed"))
		})
	})
})
