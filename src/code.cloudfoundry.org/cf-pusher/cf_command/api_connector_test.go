package cf_command_test

import (
	"errors"

	"code.cloudfoundry.org/cf-pusher/cf_command"
	"code.cloudfoundry.org/cf-pusher/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ApiConnector", func() {

	var (
		apiConnector *cf_command.ApiConnector
		fakeAdapter  *fakes.ApiCliAdapter
	)

	BeforeEach(func() {
		fakeAdapter = &fakes.ApiCliAdapter{}
		apiConnector = &cf_command.ApiConnector{
			Api:               "api.mycf.com",
			AdminUser:         "admin",
			AdminPassword:     "my-password",
			SkipSSLValidation: true,
			Adapter:           fakeAdapter,
		}
	})

	It("Sets the API URL and logs in as Admin", func() {
		err := apiConnector.Connect()
		Expect(err).NotTo(HaveOccurred())
		Expect(fakeAdapter.SetApiWithoutSslCallCount()).To(Equal(1))
		Expect(fakeAdapter.SetApiWithoutSslArgsForCall(0)).To(Equal("api.mycf.com"))
		Expect(fakeAdapter.AuthCallCount()).To(Equal(1))
		user, password := fakeAdapter.AuthArgsForCall(0)
		Expect(user).To(Equal("admin"))
		Expect(password).To(Equal("my-password"))
	})
	Context("when setting the api fails", func() {
		BeforeEach(func() {
			fakeAdapter.SetApiWithoutSslReturns(errors.New("banana"))
		})
		It("returns the error", func() {
			err := apiConnector.Connect()
			Expect(err).To(MatchError("setting api without ssl: banana"))
		})
	})
	Context("when authenticating with cf fails", func() {
		BeforeEach(func() {
			fakeAdapter.AuthReturns(errors.New("banana"))
		})
		It("returns the error", func() {
			err := apiConnector.Connect()
			Expect(err).To(MatchError("authenticating: banana"))
		})
	})

	Context("When SkipSSLValidation is false", func() {
		BeforeEach(func() {
			apiConnector.SkipSSLValidation = false
		})
		It("Sets the API URL without SSL", func() {
			err := apiConnector.Connect()
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeAdapter.SetApiWithSslCallCount()).To(Equal(1))
			Expect(fakeAdapter.SetApiWithSslArgsForCall(0)).To(Equal("api.mycf.com"))
		})
		Context("when setting the api fails", func() {
			BeforeEach(func() {
				fakeAdapter.SetApiWithSslReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				err := apiConnector.Connect()
				Expect(err).To(MatchError("setting api with ssl: banana"))
			})
		})
	})
})
