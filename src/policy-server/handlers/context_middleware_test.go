package handlers_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Context Wrapper", func() {
	var (
		request        *http.Request
		response       *httptest.ResponseRecorder
		contextWrapper *handlers.ContextWrapper

		innerHandler   *fakes.HTTPHandler
		outerHandler   http.Handler
		contextAdapter *fakes.ContextAdapter
		fakeContext    context.Context
		count          int
	)

	BeforeEach(func() {
		contextAdapter = &fakes.ContextAdapter{}
		fakeContext = context.Background()
		contextAdapter.WithTimeoutReturns(fakeContext, func() {
			count = innerHandler.ServeHTTPCallCount()
		})

		contextWrapper = &handlers.ContextWrapper{
			Duration:       5 * time.Second,
			ContextAdapter: contextAdapter,
		}

		innerHandler = &fakes.HTTPHandler{}
		outerHandler = contextWrapper.Wrap(innerHandler)

		var err error
		request, err = http.NewRequest("GET", "asdf", bytes.NewBuffer([]byte{}))
		Expect(err).NotTo(HaveOccurred())
	})

	It("wraps the handler with the timeout", func() {
		outerHandler.ServeHTTP(response, request)
		Expect(innerHandler.ServeHTTPCallCount()).To(Equal(1))
		resp, req := innerHandler.ServeHTTPArgsForCall(0)

		Expect(resp).To(Equal(response))
		Expect(req.Context()).To(Equal(fakeContext))

		Expect(contextAdapter.WithTimeoutCallCount()).To(Equal(1))
		ctx, dur := contextAdapter.WithTimeoutArgsForCall(0)
		Expect(ctx).To(Equal(request.Context()))
		Expect(dur).To(Equal(5 * time.Second))

		Expect(count).To(Equal(1))
	})
})
