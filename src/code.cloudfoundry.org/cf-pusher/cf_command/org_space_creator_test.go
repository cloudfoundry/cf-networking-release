package cf_command_test

import (
	"errors"

	"code.cloudfoundry.org/cf-pusher/cf_command"
	"code.cloudfoundry.org/cf-pusher/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OrgSpaceCreator", func() {
	var (
		orgSpaceCreator *cf_command.OrgSpaceCreator
		fakeCli         *fakes.OrgSpaceCli
		quota           cf_command.Quota
	)
	BeforeEach(func() {
		fakeCli = &fakes.OrgSpaceCli{}
		quota = cf_command.Quota{
			Name:             "some-quota",
			Memory:           "100G",
			InstanceMemory:   -1,
			Routes:           10000,
			ServiceInstances: 100,
			AppInstances:     -1,
			RoutePorts:       -1,
		}

		orgSpaceCreator = &cf_command.OrgSpaceCreator{
			Org:     "some-org",
			Space:   "some-space",
			Quota:   quota,
			Adapter: fakeCli,
		}
	})
	It("creates and targets the org and space with a custom quota", func() {
		err := orgSpaceCreator.Create()
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeCli.CreateOrgCallCount()).To(Equal(1))
		Expect(fakeCli.CreateOrgArgsForCall(0)).To(Equal("some-org"))
		Expect(fakeCli.TargetOrgCallCount()).To(Equal(1))
		Expect(fakeCli.TargetOrgArgsForCall(0)).To(Equal("some-org"))
		Expect(fakeCli.CreateSpaceCallCount()).To(Equal(1))
		spaceName, orgName := fakeCli.CreateSpaceArgsForCall(0)
		Expect(spaceName).To(Equal("some-space"))
		Expect(orgName).To(Equal("some-org"))
		Expect(fakeCli.TargetSpaceCallCount()).To(Equal(1))
		Expect(fakeCli.TargetSpaceArgsForCall(0)).To(Equal("some-space"))
		Expect(fakeCli.CreateQuotaCallCount()).To(Equal(1))
		q, m, i, r, s, a, p := fakeCli.CreateQuotaArgsForCall(0)
		Expect(q).To(Equal("some-quota"))
		Expect(m).To(Equal("100G"))
		Expect(i).To(Equal(-1))
		Expect(r).To(Equal(10000))
		Expect(s).To(Equal(100))
		Expect(a).To(Equal(-1))
		Expect(p).To(Equal(-1))
		Expect(fakeCli.SetQuotaCallCount()).To(Equal(1))
		o, q := fakeCli.SetQuotaArgsForCall(0)
		Expect(o).To(Equal("some-org"))
		Expect(q).To(Equal("some-quota"))
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

	Context("when creating the quota fails", func() {
		BeforeEach(func() {
			fakeCli.CreateQuotaReturns(errors.New("banana"))
		})
		It("returns the error", func() {
			err := orgSpaceCreator.Create()
			Expect(err).To(MatchError("creating quota: banana"))
		})
	})

	Context("when setting the quota fails", func() {
		BeforeEach(func() {
			fakeCli.SetQuotaReturns(errors.New("banana"))
		})
		It("returns the error", func() {
			err := orgSpaceCreator.Create()
			Expect(err).To(MatchError("setting quota: banana"))
		})
	})
})
