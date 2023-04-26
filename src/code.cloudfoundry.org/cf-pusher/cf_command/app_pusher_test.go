package cf_command_test

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/cf-pusher/cf_command"
	"code.cloudfoundry.org/cf-pusher/fakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AppPusher", func() {
	var (
		appPusher   *cf_command.AppPusher
		fakeAdapter *fakes.PushCLIAdapter
	)

	BeforeEach(func() {
		fakeAdapter = &fakes.PushCLIAdapter{}
		appPusher = &cf_command.AppPusher{
			Adapter:      fakeAdapter,
			Concurrency:  2,
			Directory:    "some/dir",
			ManifestPath: "some/tmp/dir/manifest.yml",

			RetryAttempts: 3,
			RetryWaitTime: time.Millisecond,
		}
	})

	Describe("Push", func() {
		BeforeEach(func() {
			appPusher.Applications = []cf_command.Application{
				{
					Name: "failed-getting-guid",
				},
			}
		})

		It("writes out the manifest and uses it", func() {
			err := appPusher.Push()
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeAdapter.PushCallCount()).To(Equal(1))
			name, dir, manifestFile := fakeAdapter.PushArgsForCall(0)
			Expect(name).To(Equal("failed-getting-guid"))
			Expect(dir).To(Equal("some/dir"))
			Expect(manifestFile).To(Equal("some/tmp/dir/manifest.yml"))
		})

		Context("when pushing an app fails", func() {
			BeforeEach(func() {
				var callCount uint32 = 0
				fakeAdapter.PushStub = func(x, y, z string) error {
					count := atomic.AddUint32(&callCount, 1)
					if count < 3 {
						return errors.New("potato")
					} else {
						return nil
					}
				}
			})

			It("retries pushing the app twice", func() {
				err := appPusher.Push()
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeAdapter.PushCallCount()).To(Equal(3))

				for pushAttempt := 0; pushAttempt < 3; pushAttempt++ {
					name, dir, manifestFile := fakeAdapter.PushArgsForCall(pushAttempt)
					Expect(name).To(Equal("failed-getting-guid"))
					Expect(dir).To(Equal("some/dir"))
					Expect(manifestFile).To(Equal("some/tmp/dir/manifest.yml"))
				}
			})
		})

		Context("when there are multiple apps", func() {
			BeforeEach(func() {
				appPusher.Applications = []cf_command.Application{}
				for i := 0; i < 10; i++ {
					app := cf_command.Application{
						Name: fmt.Sprintf("failed-getting-guid-%d", i),
					}
					appPusher.Applications = append(appPusher.Applications, app)
				}
			})

			It("calls check and then push for each app", func() {
				err := appPusher.Push()
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeAdapter.AppGuidCallCount()).To(Equal(0))
				Expect(fakeAdapter.PushCallCount()).To(Equal(10))
			})

			Context("when pushing an app fails", func() {
				BeforeEach(func() {
					fakeAdapter.PushReturns(errors.New("potato"))
				})

				It("return the error", func() {
					err := appPusher.Push()
					Expect(err).To(MatchError("potato"))
				})
			})

			Context("when pushing the last app fails", func() {
				BeforeEach(func() {
					fakeAdapter.PushStub = func(name, y, z string) error {
						if name == "failed-getting-guid-9" {
							return errors.New("potato")
						}
						return nil
					}
				})

				It("return the error", func() {
					err := appPusher.Push()
					Expect(err).To(MatchError("potato"))
					Expect(fakeAdapter.PushCallCount()).To(Equal(12))
				})
			})
		})

		Context("when SkipIfPresent is true", func() {
			BeforeEach(func() {
				appPusher.SkipIfPresent = true
				appPusher.DesiredRunningInstances = 5
				appPusher.Concurrency = 1
				appPusher.Applications = []cf_command.Application{
					{Name: "failed-getting-guid"},
					{Name: "app-to-be-skipped"},
					{Name: "not-enough-instances"},
					{Name: "failed-unmarshalling-app"},
					{Name: "failed-checking-app"},
				}
				fakeAdapter.AppGuidStub = func(name string) (string, error) {
					if name == "failed-getting-guid" {
						return "", errors.New("banana")
					} else if name == "app-to-be-skipped" {
						return "app-to-be-skipped-guid", nil
					} else if name == "not-enough-instances" {
						return "not-enough-instances-guid", nil
					} else if name == "failed-unmarshalling-app" {
						return "failed-unmarshalling-app-guid", nil
					} else if name == "failed-checking-app" {
						return "failed-checking-app-guid", nil
					}
					return "", nil
				}

				fakeAdapter.CheckAppStub = func(guid string) ([]byte, error) {
					if guid == "app-to-be-skipped-guid" {
						return []byte(`{"running_instances": 5}`), nil
					} else if guid == "not-enough-instances-guid" {
						return []byte(`{"running_instances": 3}`), nil
					} else if guid == "failed-unmarshalling-app-guid" {
						return []byte(`invalid json`), nil
					} else if guid == "failed-checking-app-guid" {
						return []byte(`{"running_instances": 5}`), errors.New("doesn't matter")
					}
					return []byte{}, errors.New("doesn't matter")
				}
			})
			It("doesn't push apps that already have running instances", func() {
				err := appPusher.Push()
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeAdapter.AppGuidCallCount()).To(Equal(5))
				Expect(fakeAdapter.CheckAppCallCount()).To(Equal(4))
				Expect(fakeAdapter.CheckAppArgsForCall(0)).To(Equal("app-to-be-skipped-guid"))
				Expect(fakeAdapter.CheckAppArgsForCall(1)).To(Equal("not-enough-instances-guid"))
				Expect(fakeAdapter.CheckAppArgsForCall(2)).To(Equal("failed-unmarshalling-app-guid"))
				Expect(fakeAdapter.CheckAppArgsForCall(3)).To(Equal("failed-checking-app-guid"))

				By("pushing apps that are not pushed")
				Expect(fakeAdapter.PushCallCount()).To(Equal(4))
				pushedApp, dir, manifest := fakeAdapter.PushArgsForCall(0)
				Expect(pushedApp).To(Equal("failed-getting-guid"))
				Expect(dir).To(Equal("some/dir"))
				Expect(manifest).To(Equal("some/tmp/dir/manifest.yml"))

				By("pushing apps that do not have enough instances running")
				pushedApp, dir, manifest = fakeAdapter.PushArgsForCall(1)
				Expect(pushedApp).To(Equal("not-enough-instances"))
				Expect(dir).To(Equal("some/dir"))
				Expect(manifest).To(Equal("some/tmp/dir/manifest.yml"))

				By("pushing apps whose summary return invalid json")
				pushedApp, dir, manifest = fakeAdapter.PushArgsForCall(2)
				Expect(pushedApp).To(Equal("failed-unmarshalling-app"))
				Expect(dir).To(Equal("some/dir"))
				Expect(manifest).To(Equal("some/tmp/dir/manifest.yml"))

				By("pushing apps that we fail to check the instances running")
				pushedApp, dir, manifest = fakeAdapter.PushArgsForCall(3)
				Expect(pushedApp).To(Equal("failed-checking-app"))
				Expect(dir).To(Equal("some/dir"))
				Expect(manifest).To(Equal("some/tmp/dir/manifest.yml"))
			})
		})
	})
})
