package cf_command_test

import (
	"errors"

	"code.cloudfoundry.org/cf-pusher/cf_command"
	"code.cloudfoundry.org/cf-pusher/fakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OrgChecker", func() {
	var (
		orgChecker *cf_command.OrgChecker
		fakeCli    *fakes.OrgCheckerCli
	)
	BeforeEach(func() {
		fakeCli = &fakes.OrgCheckerCli{}
		orgChecker = &cf_command.OrgChecker{
			Org:     "some-org",
			Adapter: fakeCli,
		}
	})

	It("targets the org", func() {
		orgExists := orgChecker.CheckOrgExists()
		Expect(orgExists).To(BeTrue())

		Expect(fakeCli.TargetOrgCallCount()).To(Equal(1))
		Expect(fakeCli.TargetOrgArgsForCall(0)).To(Equal("some-org"))
	})

	Context("when targeting the org fails", func() {
		BeforeEach(func() {
			fakeCli.TargetOrgReturns(errors.New("banana"))
		})
		It("returns false", func() {
			orgExists := orgChecker.CheckOrgExists()
			Expect(orgExists).To(BeFalse())
		})
	})
})
