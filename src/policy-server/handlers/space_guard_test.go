package handlers_test

import (
	"policy-server/fakes"
	"policy-server/handlers"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SpaceGuard", func() {
	var (
		spaceGuard *handlers.SpaceGuard
		logger     *lagertest.TestLogger
		client     *fakes.Client
	)

	BeforeEach(func() {
		spaceGuard = &handlers.SpaceGuard{
			Logger: logger,
			Client: client,
		}
		client.GetSpaceGuidsReturns([]string{"space-guid-1", "space-guid-2", "space-guid-3"})
	})

	Describe("CheckRequest", func() {
		It("looks up the space guids for all apps in the request", func() {
			// spaceGuard.CheckRequest(req, token)

		})

		It("looks up the space guids for all apps in the request", func() {

		})

		It("checks that the user has SpaceDeveloper role in all spaces", func() {

		})

		It("returns nil if the user has access to all apps in request", func() {

		})

		Context("when the user cannot access one or more apps", func() {
			It("returns an useful error", func() {

			})
		})

		Context("when more than one page of apps is in the request", func() {
			It("returns an useful error", func() {

			})
		})

	})
})
