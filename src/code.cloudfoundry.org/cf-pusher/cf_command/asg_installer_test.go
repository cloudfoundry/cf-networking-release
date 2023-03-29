package cf_command_test

import (
	"errors"

	"code.cloudfoundry.org/cf-pusher/cf_command"
	"code.cloudfoundry.org/cf-pusher/fakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AsgInstaller", func() {
	var (
		fakeAdapter  *fakes.SecurityGroupInstallationCLIAdapter
		asgInstaller *cf_command.ASGInstaller
	)

	BeforeEach(func() {
		fakeAdapter = &fakes.SecurityGroupInstallationCLIAdapter{}
		asgInstaller = &cf_command.ASGInstaller{Adapter: fakeAdapter}
	})

	Describe("InstallASG", func() {
		It("deletes the existing ASG", func() {
			err := asgInstaller.InstallASG("some-asg-name", "some-asg-file-path", "some-org", "some-space")
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeAdapter.DeleteSecurityGroupCallCount()).To(Equal(1))
			Expect(fakeAdapter.DeleteSecurityGroupArgsForCall(0)).To(Equal("some-asg-name"))
		})

		It("creates the (new) security group with the given name", func() {
			err := asgInstaller.InstallASG("some-asg-name", "some-asg-file-path", "some-org", "some-space")
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeAdapter.CreateSecurityGroupCallCount()).To(Equal(1))
			asgName, asgBody := fakeAdapter.CreateSecurityGroupArgsForCall(0)
			Expect(asgName).To(Equal("some-asg-name"))
			Expect(asgBody).To(Equal("some-asg-file-path"))
		})

		It("binds the security group to the org and space", func() {
			err := asgInstaller.InstallASG("some-asg-name", "some-asg-file-path", "some-org", "some-space")
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeAdapter.BindSecurityGroupCallCount()).To(Equal(1))
			asgName, orgName, spaceName := fakeAdapter.BindSecurityGroupArgsForCall(0)
			Expect(asgName).To(Equal("some-asg-name"))
			Expect(orgName).To(Equal("some-org"))
			Expect(spaceName).To(Equal("some-space"))
		})

		Context("when deleting the security group fails", func() {
			BeforeEach(func() {
				fakeAdapter.DeleteSecurityGroupReturns(errors.New("banana"))
			})
			It("returns a meaningful error", func() {
				err := asgInstaller.InstallASG("some-asg-name", "some-asg-file-path", "some-org", "some-space")
				Expect(err).To(MatchError("deleting security group: banana"))
			})
		})

		Context("when creating the security group fails", func() {
			BeforeEach(func() {
				fakeAdapter.CreateSecurityGroupReturns(errors.New("banana"))
			})
			It("returns a meaningful error", func() {
				err := asgInstaller.InstallASG("some-asg-name", "some-asg-file-path", "some-org", "some-space")
				Expect(err).To(MatchError("creating security group: banana"))
			})
		})

		Context("when binding the security group fails", func() {
			BeforeEach(func() {
				fakeAdapter.BindSecurityGroupReturns(errors.New("banana"))
			})
			It("returns a meaningful error", func() {
				err := asgInstaller.InstallASG("some-asg-name", "some-asg-file-path", "some-org", "some-space")
				Expect(err).To(MatchError("binding security group: banana"))
			})
		})
	})

	Describe("InstallGlobalASG", func() {
		It("deletes the existing ASG", func() {
			err := asgInstaller.InstallGlobalASG("some-asg-name", "some-asg-file-path")
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeAdapter.DeleteSecurityGroupCallCount()).To(Equal(1))
			Expect(fakeAdapter.DeleteSecurityGroupArgsForCall(0)).To(Equal("some-asg-name"))
		})

		It("creates the (new) security group with the given name", func() {
			err := asgInstaller.InstallGlobalASG("some-asg-name", "some-asg-file-path")
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeAdapter.CreateSecurityGroupCallCount()).To(Equal(1))
			asgName, asgBody := fakeAdapter.CreateSecurityGroupArgsForCall(0)
			Expect(asgName).To(Equal("some-asg-name"))
			Expect(asgBody).To(Equal("some-asg-file-path"))
		})

		It("binds the security group globally", func() {
			err := asgInstaller.InstallGlobalASG("some-asg-name", "some-asg-file-path")
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeAdapter.BindGlobalRunningSecurityGroupCallCount()).To(Equal(1))
			asgName := fakeAdapter.BindGlobalRunningSecurityGroupArgsForCall(0)
			Expect(asgName).To(Equal("some-asg-name"))
		})

		Context("when deleting the security group fails", func() {
			BeforeEach(func() {
				fakeAdapter.DeleteSecurityGroupReturns(errors.New("banana"))
			})
			It("returns a meaningful error", func() {
				err := asgInstaller.InstallGlobalASG("some-asg-name", "some-asg-file-path")
				Expect(err).To(MatchError("deleting security group: banana"))
			})
		})

		Context("when creating the security group fails", func() {
			BeforeEach(func() {
				fakeAdapter.CreateSecurityGroupReturns(errors.New("banana"))
			})
			It("returns a meaningful error", func() {
				err := asgInstaller.InstallGlobalASG("some-asg-name", "some-asg-file-path")
				Expect(err).To(MatchError("creating security group: banana"))
			})
		})

		Context("when binding the security group fails", func() {
			BeforeEach(func() {
				fakeAdapter.BindGlobalRunningSecurityGroupReturns(errors.New("banana"))
			})
			It("returns a meaningful error", func() {
				err := asgInstaller.InstallGlobalASG("some-asg-name", "some-asg-file-path")
				Expect(err).To(MatchError("binding global running security group: banana"))
			})
		})
	})
})
