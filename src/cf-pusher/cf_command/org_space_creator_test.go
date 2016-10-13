package cf_command_test

import (
	"cf-pusher/cf_command"
	"cf-pusher/fakes"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OrgSpaceCreator", func() {
	var (
		orgSpaceCreator *cf_command.OrgSpaceCreator
		fakeCli         *fakes.OrgSpaceCli
	)
	BeforeEach(func() {
		fakeCli = &fakes.OrgSpaceCli{}
		orgSpaceCreator = &cf_command.OrgSpaceCreator{
			Org:     "some-org",
			Space:   "some-space",
			Adapter: fakeCli,
		}
	})
	It("creates and targets the org and space", func() {
		err := orgSpaceCreator.Create()
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeCli.CreateOrgCallCount()).To(Equal(1))
		Expect(fakeCli.CreateOrgArgsForCall(0)).To(Equal("some-org"))
		Expect(fakeCli.TargetOrgCallCount()).To(Equal(1))
		Expect(fakeCli.TargetOrgArgsForCall(0)).To(Equal("some-org"))
		Expect(fakeCli.CreateSpaceCallCount()).To(Equal(1))
		Expect(fakeCli.CreateSpaceArgsForCall(0)).To(Equal("some-space"))
		Expect(fakeCli.TargetSpaceCallCount()).To(Equal(1))
		Expect(fakeCli.TargetSpaceArgsForCall(0)).To(Equal("some-space"))
	})

	Context("when creating the org fails", func() {
		BeforeEach(func() {
			fakeCli.CreateOrgReturns(errors.New("banana"))
		})
		It("returns the error", func() {
			err := orgSpaceCreator.Create()
			Expect(err).To(MatchError("creating org: banana"))
		})
	})

	Context("when creating the space fails", func() {
		BeforeEach(func() {
			fakeCli.CreateSpaceReturns(errors.New("banana"))
		})
		It("returns the error", func() {
			err := orgSpaceCreator.Create()
			Expect(err).To(MatchError("creating space: banana"))
		})
	})

	Context("when targeting the org fails", func() {
		BeforeEach(func() {
			fakeCli.TargetOrgReturns(errors.New("banana"))
		})
		It("returns the error", func() {
			err := orgSpaceCreator.Create()
			Expect(err).To(MatchError("targeting org: banana"))
		})
	})

	Context("when targeting the space fails", func() {
		BeforeEach(func() {
			fakeCli.TargetSpaceReturns(errors.New("banana"))
		})
		It("returns the error", func() {
			err := orgSpaceCreator.Create()
			Expect(err).To(MatchError("targeting space: banana"))
		})
	})
})
