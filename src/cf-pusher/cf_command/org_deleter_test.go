package cf_command_test

import (
	"cf-pusher/cf_command"
	"cf-pusher/fakes"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OrgDeleter", func() {
	var (
		orgDeleter *cf_command.OrgDeleter
		fakeCli    *fakes.OrgDeleterCli
	)
	BeforeEach(func() {
		fakeCli = &fakes.OrgDeleterCli{}
		orgDeleter = &cf_command.OrgDeleter{
			Org:     "some-org",
			Adapter: fakeCli,
		}
	})

	It("deletes the org", func() {
		err := orgDeleter.Delete()
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeCli.DeleteOrgCallCount()).To(Equal(1))
		Expect(fakeCli.DeleteOrgArgsForCall(0)).To(Equal("some-org"))
	})

	Context("when deleting the org fails", func() {
		BeforeEach(func() {
			fakeCli.DeleteOrgReturns(errors.New("banana"))
		})
		It("returns the error", func() {
			err := orgDeleter.Delete()
			Expect(err).To(MatchError("banana"))
		})
	})
})
