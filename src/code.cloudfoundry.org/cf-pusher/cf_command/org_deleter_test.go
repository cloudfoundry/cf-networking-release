package cf_command_test

import (
	"errors"

	"code.cloudfoundry.org/cf-pusher/cf_command"
	"code.cloudfoundry.org/cf-pusher/fakes"

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
			Quota:   cf_command.Quota{Name: "some-quota"},
			Adapter: fakeCli,
		}
	})

	It("deletes the org and quota", func() {
		err := orgDeleter.Delete()
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeCli.DeleteOrgCallCount()).To(Equal(1))
		Expect(fakeCli.DeleteOrgArgsForCall(0)).To(Equal("some-org"))
		Expect(fakeCli.DeleteQuotaCallCount()).To(Equal(1))
		Expect(fakeCli.DeleteQuotaArgsForCall(0)).To(Equal("some-quota"))
	})

	Context("when deleting the org fails", func() {
		BeforeEach(func() {
			fakeCli.DeleteOrgReturns(errors.New("banana"))
		})
		It("returns a meaningful error", func() {
			err := orgDeleter.Delete()
			Expect(err).To(MatchError("deleting org: banana"))
		})
	})

	Context("when deleting the quota fails", func() {
		BeforeEach(func() {
			fakeCli.DeleteQuotaReturns(errors.New("banana"))
		})
		It("returns a meaningful error", func() {
			err := orgDeleter.Delete()
			Expect(err).To(MatchError("deleting quota: banana"))
		})
	})
})
