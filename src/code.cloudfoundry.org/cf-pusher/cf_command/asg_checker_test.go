package cf_command_test

import (
	"errors"

	"code.cloudfoundry.org/cf-pusher/cf_command"
	"code.cloudfoundry.org/cf-pusher/fakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AsgChecker", func() {
	var (
		fakeAdapter *fakes.SecurityGroupCLIAdapter
		expectedASG string
		asgChecker  *cf_command.ASGChecker
	)
	BeforeEach(func() {
		fakeAdapter = &fakes.SecurityGroupCLIAdapter{}
		expectedASG = `[{ "foo": "bar" }]`
		asgChecker = &cf_command.ASGChecker{
			Adapter: fakeAdapter,
		}
		fakeAdapter.SecurityGroupReturns(`[{"foo":"bar"}]`, nil)
	})

	Describe("CheckASGs", func() {
		Context("when the expected ASG matches what exists (whitespace insensitive)", func() {
			It("succeeds", func() {
				err := asgChecker.CheckASG("some-asg-name", expectedASG)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeAdapter.SecurityGroupCallCount()).To(Equal(1))
				Expect(fakeAdapter.SecurityGroupArgsForCall(0)).To(Equal("some-asg-name"))
			})
		})

		Context("when the existing ASG does not match the expected one", func() {
			BeforeEach(func() {
				fakeAdapter.SecurityGroupReturns(`[{ "foo": "baz" }]`, nil)
			})

			It("returns a meaningful error", func() {
				err := asgChecker.CheckASG("some-asg-name", expectedASG)
				Expect(err).To(MatchError("security group mismatch"))
			})
		})

		Context("when the cf cli call returns an error", func() {
			BeforeEach(func() {
				fakeAdapter.SecurityGroupReturns("", errors.New("banana"))
			})

			It("wraps and returns the error", func() {
				err := asgChecker.CheckASG("some-asg-name", expectedASG)
				Expect(err).To(MatchError("getting security group: banana"))
			})
		})

		Context("when the expectedASG is invalid JSON", func() {
			BeforeEach(func() {
				expectedASG = "foo"
			})

			It("wraps and returns the error", func() {
				err := asgChecker.CheckASG("some-asg-name", expectedASG)
				Expect(err).To(MatchError("expected ASG is not valid JSON: foo"))
			})
		})

		Context("when the cli returns bad invalid JSON", func() {
			BeforeEach(func() {
				fakeAdapter.SecurityGroupReturns("foo", nil)
			})

			It("wraps and returns the error", func() {
				err := asgChecker.CheckASG("some-asg-name", expectedASG)
				Expect(err).To(MatchError("actual ASG is not valid JSON: foo"))
			})
		})

	})
})
