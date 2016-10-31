package cf_command_test

import (
	"cf-pusher/cf_command"
	"cf-pusher/fakes"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AppChecker", func() {
	var (
		appChecker  *cf_command.AppChecker
		fakeAdapter *fakes.CheckCLIAdapter
		appSpec     map[string]int
	)
	BeforeEach(func() {
		fakeAdapter = &fakes.CheckCLIAdapter{}
		appChecker = &cf_command.AppChecker{
			Org:     "some-org-name",
			Adapter: fakeAdapter,
		}
	})
	Describe("CheckApps", func() {
		BeforeEach(func() {
			appChecker.Applications = []cf_command.Application{
				{
					Name:      "some-name-1",
					Directory: "some/dir",
				},
			}
			appSpec = map[string]int{}
			appSpec["some-name-1"] = 2
			fakeAdapter.AppGuidReturns("some-guid-1", nil)
			str := `{ "guid": "some-guid-1", "name": "scale-tick-1", "running_instances": 2, "instances": 2, "state": "STARTED"}`
			fakeAdapter.CheckAppReturns([]byte(str), nil)
			fakeAdapter.OrgGuidReturns("some-org-guid", nil)
			fakeAdapter.AppCountReturns(1, nil)
		})
		It("when the app is in state running", func() {
			err := appChecker.CheckApps(appSpec)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeAdapter.AppGuidCallCount()).To(Equal(1))
			Expect(fakeAdapter.AppGuidArgsForCall(0)).To(Equal("some-name-1"))

			Expect(fakeAdapter.CheckAppCallCount()).To(Equal(1))
			Expect(fakeAdapter.CheckAppArgsForCall(0)).To(Equal("some-guid-1"))

			Expect(fakeAdapter.OrgGuidCallCount()).To(Equal(1))
			Expect(fakeAdapter.OrgGuidArgsForCall(0)).To(Equal("some-org-name"))
			Expect(fakeAdapter.AppCountCallCount()).To(Equal(1))
			Expect(fakeAdapter.AppCountArgsForCall(0)).To(Equal("some-org-guid"))
		})

		Context("when an app is not running the specified number of instances", func() {
			BeforeEach(func() {
				appSpec["some-name-1"] = 1
			})
			It("returns an error", func() {
				err := appChecker.CheckApps(appSpec)
				Expect(err).To(MatchError("checking app some-name-1: not running desired instances, running: 2 desired: 1"))
			})
		})

		Context("when the app name is not in the app spec", func() {
			BeforeEach(func() {
				appChecker.Applications = append(appChecker.Applications, cf_command.Application{
					Name:      "banana",
					Directory: "some/dir",
				})
				fakeAdapter.AppCountReturns(2, nil)
			})
			It("returns a helpful error", func() {
				err := appChecker.CheckApps(appSpec)
				Expect(err).To(MatchError("checking app banana: not found in app spec"))
			})
		})

		Context("when org guid fails", func() {
			BeforeEach(func() {
				fakeAdapter.OrgGuidReturns("", errors.New("potato"))
			})
			It("returns a meaningful error", func() {
				err := appChecker.CheckApps(appSpec)
				Expect(err).To(MatchError("checking org guid some-org-name: potato"))
			})
		})

		Context("when app count fails", func() {
			BeforeEach(func() {
				fakeAdapter.AppCountReturns(-1, errors.New("potato"))
			})
			It("returns a meaningful error", func() {
				err := appChecker.CheckApps(appSpec)
				Expect(err).To(MatchError("checking app counts: potato"))
			})
		})

		Context("when app count does not match", func() {
			BeforeEach(func() {
				fakeAdapter.AppCountReturns(2, nil)
			})
			It("returns a meaningful error", func() {
				err := appChecker.CheckApps(appSpec)
				Expect(err).To(MatchError("app count 2 does not match 1"))
			})
		})

		Context("when check app guid fails", func() {
			BeforeEach(func() {
				fakeAdapter.AppGuidReturns("", errors.New("potato"))
			})
			It("returns a meaningful error", func() {
				err := appChecker.CheckApps(appSpec)
				Expect(err).To(MatchError("checking app guid some-name-1: potato"))
			})
		})
		Context("when check app fails", func() {
			BeforeEach(func() {
				fakeAdapter.CheckAppReturns(nil, errors.New("potato"))
			})
			It("returns a meaningful error", func() {
				err := appChecker.CheckApps(appSpec)
				Expect(err).To(MatchError("checking app some-name-1: potato"))
			})
		})
		Context("when the json is malformed", func() {
			BeforeEach(func() {
				str := `{ "guid": "some-guid-1", "name": "scale-tick-1"`
				fakeAdapter.CheckAppReturns([]byte(str), nil)
			})
			It("returns a meaningul error", func() {
				err := appChecker.CheckApps(appSpec)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when an error response is returned", func() {
			BeforeEach(func() {
				str := ` { "code": 100004,
						   "description": "The app could not be found: guid",
						   "error_code": "CF-AppNotFound"
						}`
				fakeAdapter.CheckAppReturns([]byte(str), nil)
			})
			It("returns a meaningul error", func() {
				err := appChecker.CheckApps(appSpec)
				Expect(err).To(MatchError("checking app some-name-1: no instances are running"))
			})
		})

		Context("when response is unexpected json or no instances are running", func() {
			BeforeEach(func() {
				str := `{}`
				fakeAdapter.CheckAppReturns([]byte(str), nil)
			})
			It("returns a meaningul error", func() {
				err := appChecker.CheckApps(appSpec)
				Expect(err).To(MatchError("checking app some-name-1: no instances are running"))
			})
		})

		Context("when one app is not running", func() {
			BeforeEach(func() {
				str := `{ "guid": "some-guid-1", "name": "scale-tick-1", "running_instances": 1, "instances": 2, "state": "STARTED"}`
				fakeAdapter.CheckAppReturns([]byte(str), nil)
			})
			It("returns a meaningul error", func() {
				err := appChecker.CheckApps(appSpec)
				Expect(err).To(MatchError("checking app some-name-1: not all instances are running"))
			})

		})
	})
})
