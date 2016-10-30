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
			Adapter:           fakeAdapter,
			ManifestGenerator: fakeManifestGenerator,
			Concurrency:       2,
		}
	})
	Describe("Push", func() {
		Context("when the manifest on the app is nil", func() {
			BeforeEach(func() {
				appPusher.Applications = []cf_command.Application{
					{
						Name:      "some-name",
						Directory: "some/dir",
					},
				}
			})
			It("uses the manifest in the app directory", func() {
				err := appPusher.Push()
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeAdapter.PushCallCount()).To(Equal(1))
				name, dir, manifestFile := fakeAdapter.PushArgsForCall(0)
				Expect(name).To(Equal("some-name"))
				Expect(dir).To(Equal("some/dir"))
				Expect(manifestFile).To(Equal("some/dir/manifest.yml"))
			})
			Context("when pushing the app fails", func() {
				BeforeEach(func() {
					fakeAdapter.PushReturns(errors.New("potato"))
				})
				It("returns the error", func() {
					err := appPusher.Push()
					Expect(err).To(MatchError("potato"))
				})
			})
		})
		Context("when the manifest is not nil", func() {
			type manifest struct {
				SomeProperty string
			}
			var manifestStruct manifest

			BeforeEach(func() {
				manifestStruct = manifest{SomeProperty: "value"}
				fakeManifestGenerator.GenerateReturns("some/tmp/dir/manifest.yml", nil)
				appPusher.Applications = []cf_command.Application{
					{
						Name:      "some-name",
						Directory: "some/dir",
						Manifest:  manifestStruct,
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
				Expect(fakeManifestGenerator.GenerateCallCount()).To(Equal(1))
				Expect(fakeManifestGenerator.GenerateArgsForCall(0)).To(Equal(manifestStruct))
			})
			Context("when generating the manifest fails", func() {
				BeforeEach(func() {
					fakeManifestGenerator.GenerateReturns("", errors.New("potato"))
				})
				It("return the error", func() {
					err := appPusher.Push()
					Expect(err).To(MatchError("potato"))
				})
			})
		})
		Context("when there are multiple apps", func() {
			type manifest struct {
				SomeProperty string
			}
			var manifestStruct manifest

			BeforeEach(func() {
				manifestStruct = manifest{SomeProperty: "value"}
				fakeManifestGenerator.GenerateReturns("some/tmp/dir/manifest.yml", nil)
				appPusher.Applications = []cf_command.Application{}
				for i := 0; i < 10; i++ {
					app := cf_command.Application{
						Name:      "some-name",
						Directory: "some/dir",
						Manifest:  manifestStruct,
					}
					appPusher.Applications = append(appPusher.Applications, app)
				}
			})
			It("writes out the manifest and uses it", func() {
				err := appPusher.Push()
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeAdapter.PushCallCount()).To(Equal(10))
				Expect(fakeManifestGenerator.GenerateCallCount()).To(Equal(10))
			})
			Context("when generating the manifest fails", func() {
				BeforeEach(func() {
					fakeManifestGenerator.GenerateReturns("", errors.New("potato"))
				})
				It("return the error", func() {
					err := appPusher.Push()
					Expect(err).To(MatchError("potato"))
				})
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
