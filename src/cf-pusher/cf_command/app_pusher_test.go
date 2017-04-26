package cf_command_test

import (
	"cf-pusher/cf_command"
	"cf-pusher/fakes"
	"errors"
	"sync/atomic"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AppPusher", func() {
	var (
		appPusher             *cf_command.AppPusher
		fakeAdapter           *fakes.PushCLIAdapter
		fakeManifestGenerator *fakes.ManifestGenerator
	)
	BeforeEach(func() {
		fakeAdapter = &fakes.PushCLIAdapter{}
		fakeManifestGenerator = &fakes.ManifestGenerator{}
		appPusher = &cf_command.AppPusher{
			Adapter:      fakeAdapter,
			Concurrency:  2,
			Directory:    "some/dir",
			ManifestPath: "some/tmp/dir/manifest.yml",
		}
	})
	Describe("Push", func() {
		BeforeEach(func() {
			appPusher.Applications = []cf_command.Application{
				{
					Name: "some-name",
				},
			}
		})
		It("writes out the manifest and uses it", func() {
			err := appPusher.Push()
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeAdapter.PushCallCount()).To(Equal(1))
			name, dir, manifestFile := fakeAdapter.PushArgsForCall(0)
			Expect(name).To(Equal("some-name"))
			Expect(dir).To(Equal("some/dir"))
			Expect(manifestFile).To(Equal("some/tmp/dir/manifest.yml"))
		})
		Context("when there are multiple apps", func() {
			BeforeEach(func() {
				appPusher.Applications = []cf_command.Application{}
				for i := 0; i < 10; i++ {
					app := cf_command.Application{
						Name: "some-name",
					}
					appPusher.Applications = append(appPusher.Applications, app)
				}
			})
			It("calls push for each app", func() {
				err := appPusher.Push()
				Expect(err).NotTo(HaveOccurred())
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
					var callCount uint32 = 0
					fakeAdapter.PushStub = func(x, y, z string) error {
						count := atomic.AddUint32(&callCount, 1)
						if count == 10 {
							return errors.New("potato")
						}
						return nil
					}
				})
				It("return the error", func() {
					err := appPusher.Push()
					Expect(err).To(MatchError("potato"))
				})
			})
		})
	})
})
