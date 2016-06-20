package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	lfakes "lib/fakes"
	"lib/testsupport"
	"net/http"
	"net/http/httptest"
	"policy-server/fakes"
	"policy-server/handlers"
	"policy-server/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("PoliciesCreate", func() {
	var (
		requestJSON     string
		request         *http.Request
		handler         *handlers.PoliciesCreate
		resp            *httptest.ResponseRecorder
		fakeStore       *fakes.Store
		logger          *lagertest.TestLogger
		fakeUnmarshaler *lfakes.Unmarshaler
	)

	BeforeEach(func() {
		var err error
		requestJSON = `{"policies": [
			{
				"source": {
					"id": "some-app-guid"
				},
				"destination": {
					"id": "some-other-app-guid",
					"protocol": "tcp",
					"port": 8080
				}
			},
			{
				"source": {
					"id": "another-app-guid"
				},
				"destination": {
					"id": "some-other-app-guid",
					"protocol": "udp",
					"port": 1234
				}
			}
        ]}`
		request, err = http.NewRequest("POST", "/networking/v0/external/policies", bytes.NewBuffer([]byte(requestJSON)))
		Expect(err).NotTo(HaveOccurred())

		fakeStore = &fakes.Store{}
		logger = lagertest.NewTestLogger("test")
		fakeUnmarshaler = &lfakes.Unmarshaler{}
		fakeUnmarshaler.UnmarshalStub = json.Unmarshal
		handler = &handlers.PoliciesCreate{
			Logger:      logger,
			Store:       fakeStore,
			Unmarshaler: fakeUnmarshaler,
		}
		resp = httptest.NewRecorder()
	})

	It("persists a new policy rule", func() {
		expectedPolicies := []models.Policy{{
			Source: models.Source{"some-app-guid"},
			Destination: models.Destination{
				ID:       "some-other-app-guid",
				Protocol: "tcp",
				Port:     8080,
			},
		}, {
			Source: models.Source{"another-app-guid"},
			Destination: models.Destination{
				ID:       "some-other-app-guid",
				Protocol: "udp",
				Port:     1234,
			},
		}}

		handler.ServeHTTP(resp, request)

		Expect(fakeUnmarshaler.UnmarshalCallCount()).To(Equal(1))
		bodyBytes, _ := fakeUnmarshaler.UnmarshalArgsForCall(0)
		Expect(bodyBytes).To(Equal([]byte(requestJSON)))
		Expect(fakeStore.CreateCallCount()).To(Equal(1))
		Expect(fakeStore.CreateArgsForCall(0)).To(Equal(expectedPolicies))
		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(MatchJSON("{}"))
	})

	Context("when the store Create call returns an error", func() {
		BeforeEach(func() {
			fakeStore.CreateReturns(errors.New("banana"))
		})

		It("sets a 500 error code, and returns a generic error", func() {
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "database create failed"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request)
			Expect(logger).To(gbytes.Say("store-create-failed.*banana"))
		})
	})

	Context("when the policies list is empty", func() {
		BeforeEach(func() {
			request.Body = ioutil.NopCloser(bytes.NewReader([]byte(`{"policies":[]}`)))
		})

		It("returns a descriptive error", func() {
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "missing policies"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request)
			Expect(logger).To(gbytes.Say("missing policies"))
		})
	})

	Context("when the destination port field is empty", func() {
		BeforeEach(func() {
			request.Body = ioutil.NopCloser(bytes.NewReader([]byte(`{"policies":[
			{
				"source": {
					"id": "some-app-guid"
				},
				"destination": {
					"id": "some-other-app-guid",
					"protocol": "tcp"
				}
			}
			]}`)))
		})
		It("returns a descriptive error", func() {
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "missing destination port"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request)
			Expect(logger).To(gbytes.Say("missing destination port"))
		})
	})

	Context("when the destination protocol field is empty", func() {
		BeforeEach(func() {
			request.Body = ioutil.NopCloser(bytes.NewReader([]byte(`{"policies":[
			{
				"source": {
					"id": "some-app-guid"
				},
				"destination": {
					"id": "some-other-app-guid",
					"port": 8080
				}
			}
			]}`)))
		})
		It("returns a descriptive error", func() {
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "missing destination protocol"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request)
			Expect(logger).To(gbytes.Say("missing destination protocol"))
		})
	})

	Context("when the destination id is missing", func() {
		BeforeEach(func() {
			request.Body = ioutil.NopCloser(bytes.NewReader([]byte(`{"policies":[
			{
				"source": {
					"id": "some-app-guid"
				},
				"destination": {
					"protocol": "tcp",
					"port": 8080
				}
			}
			]}`)))
		})
		It("returns a descriptive error", func() {
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "missing destination id"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request)
			Expect(logger).To(gbytes.Say("missing destination id"))
		})
	})

	Context("when the source id is missing", func() {
		BeforeEach(func() {
			request.Body = ioutil.NopCloser(bytes.NewReader([]byte(`{"policies":[
			{
				"source": {
				},
				"destination": {
					"id": "some-other-app-guid",
					"protocol": "tcp",
					"port": 8080
				}
			}
			]}`)))
		})
		It("returns a descriptive error", func() {
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "missing source id"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request)
			Expect(logger).To(gbytes.Say("missing source id"))
		})
	})

	Context("when there are errors reading the body bytes", func() {
		BeforeEach(func() {
			request.Body = ioutil.NopCloser(&testsupport.BadReader{})
		})

		It("returns a descriptive error", func() {
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "invalid request body format passed to API should be JSON"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request)
			Expect(logger).To(gbytes.Say("body-read-failed.*banana"))
		})
	})

	Context("when there are errors in the request body formatting", func() {
		BeforeEach(func() {
			request.Body = ioutil.NopCloser(bytes.NewReader([]byte(`{"policies":{}}`)))
		})

		It("returns a descriptive error", func() {
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "invalid values passed to API"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request)
			Expect(logger).To(gbytes.Say("unmarshal-failed.*json: cannot unmarshal"))
		})
	})
})
