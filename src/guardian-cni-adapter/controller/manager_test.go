package controller_test

import (
	"errors"

	"guardian-cni-adapter/controller"
	"guardian-cni-adapter/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager", func() {
	var (
		manager       *controller.Manager
		cniController *fakes.CNIController
		mounter       *fakes.Mounter
	)

	BeforeEach(func() {
		mounter = &fakes.Mounter{}
		cniController = &fakes.CNIController{}
		manager = &controller.Manager{
			CNIController: cniController,
			Mounter:       mounter,
			BindMountRoot: "/some/fake/path",
		}
	})

	Describe("Up", func() {
		It("should ensure that the netNS is mounted to the provided path", func() {
			Expect(manager.Up(42, "some-container-handle", "some-network-spec")).To(Succeed())
			Expect(mounter.IdempotentlyMountCallCount()).To(Equal(1))

			source, target := mounter.IdempotentlyMountArgsForCall(0)
			Expect(source).To(Equal("/proc/42/ns/net"))
			Expect(target).To(Equal("/some/fake/path/some-container-handle"))
		})

		It("should call CNI Up, passing in the bind-mounted path to the net ns", func() {
			Expect(manager.Up(42, "some-container-handle", "some-network-spec")).To(Succeed())
			Expect(cniController.UpCallCount()).To(Equal(1))
			namespacePath, handle, spec := cniController.UpArgsForCall(0)
			Expect(namespacePath).To(Equal("/some/fake/path/some-container-handle"))
			Expect(handle).To(Equal("some-container-handle"))
			Expect(spec).To(Equal("some-network-spec"))
		})

		Context("when missing args", func() {
			It("should return a friendly error", func() {
				err := manager.Up(0, "some-container-handle", "some-network-spec")
				Expect(err).To(MatchError("up missing pid"))

				err = manager.Up(42, "", "some-network-spec")
				Expect(err).To(MatchError("up missing container handle"))
			})
		})

		Context("when missing the network spec", func() {
			It("should succeed", func() {
				err := manager.Up(42, "some-container-handle", "")
				Expect(err).NotTo(HaveOccurred())
				Expect(cniController.UpCallCount()).To(Equal(1))
				namespacePath, handle, spec := cniController.UpArgsForCall(0)
				Expect(namespacePath).To(Equal("/some/fake/path/some-container-handle"))
				Expect(handle).To(Equal("some-container-handle"))
				Expect(spec).To(BeEmpty())
			})
		})

		Context("when things fail", func() {
			Context("when the mounter fails", func() {
				It("should return the error", func() {
					mounter.IdempotentlyMountReturns(errors.New("boom"))
					err := manager.Up(42, "some-container-handle", "some-network-spec")
					Expect(err).To(MatchError("failed mounting /proc/42/ns/net to /some/fake/path/some-container-handle: boom"))
				})
			})

			Context("when the cni Up fails", func() {
				It("should return the error", func() {
					cniController.UpReturns(errors.New("bang"))
					err := manager.Up(42, "some-container-handle", "some-network-spec")
					Expect(err).To(MatchError("cni up failed: bang"))
				})
			})
		})
	})

	Describe("Down", func() {
		It("should ensure that the netNS is unmounted", func() {
			Expect(manager.Down("some-container-handle", "some-network-spec")).To(Succeed())
			Expect(mounter.RemoveMountCallCount()).To(Equal(1))

			Expect(mounter.RemoveMountArgsForCall(0)).To(Equal("/some/fake/path/some-container-handle"))
		})

		It("should call CNI Down, passing in the bind-mounted path to the net ns", func() {
			Expect(manager.Down("some-container-handle", "some-network-spec")).To(Succeed())
			Expect(cniController.DownCallCount()).To(Equal(1))
			namespacePath, handle, spec := cniController.DownArgsForCall(0)
			Expect(namespacePath).To(Equal("/some/fake/path/some-container-handle"))
			Expect(handle).To(Equal("some-container-handle"))
			Expect(spec).To(Equal("some-network-spec"))
		})

		Context("when missing args", func() {
			It("should return a friendly error", func() {
				err := manager.Down("", "")
				Expect(err).To(MatchError("down missing container handle"))
			})
		})

		Context("when things fail", func() {
			Context("when the mounter fails", func() {
				It("should return the error", func() {
					mounter.RemoveMountReturns(errors.New("boom"))
					err := manager.Down("some-container-handle", "some-network-spec")
					Expect(err).To(MatchError("failed removing mount /some/fake/path/some-container-handle: boom"))
				})
			})

			Context("when the cni Down fails", func() {
				It("should return the error", func() {
					cniController.DownReturns(errors.New("bang"))
					err := manager.Down("some-container-handle", "some-network-spec")
					Expect(err).To(MatchError("cni down failed: bang"))
				})
			})
		})
	})
})
