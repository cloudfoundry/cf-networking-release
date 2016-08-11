package controller_test

import (
	"errors"
	"net"

	"guardian-cni-adapter/controller"
	"guardian-cni-adapter/fakes"

	"github.com/containernetworking/cni/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager", func() {
	var (
		manager                 *controller.Manager
		cniController           *fakes.CNIController
		mounter                 *fakes.Mounter
		encodedGardenProperties string
	)

	BeforeEach(func() {
		mounter = &fakes.Mounter{}
		cniController = &fakes.CNIController{}
		cniController.UpReturns(&types.Result{
			IP4: &types.IPConfig{
				IP: net.IPNet{
					IP:   net.ParseIP("169.254.1.2"),
					Mask: net.IPv4Mask(255, 255, 255, 0),
				},
			},
		}, nil)
		manager = &controller.Manager{
			CNIController: cniController,
			Mounter:       mounter,
			BindMountRoot: "/some/fake/path",
		}
		encodedGardenProperties = `{ "app_id": "some-group-id" }`
	})

	Describe("Up", func() {
		It("should ensure that the netNS is mounted to the provided path", func() {
			_, err := manager.Up(42, "some-container-handle", encodedGardenProperties)
			Expect(err).NotTo(HaveOccurred())

			Expect(mounter.IdempotentlyMountCallCount()).To(Equal(1))
			source, target := mounter.IdempotentlyMountArgsForCall(0)
			Expect(source).To(Equal("/proc/42/ns/net"))
			Expect(target).To(Equal("/some/fake/path/some-container-handle"))
		})

		It("should return the IP address in the CNI result as a property", func() {
			properties, err := manager.Up(42, "some-container-handle", encodedGardenProperties)
			Expect(err).NotTo(HaveOccurred())

			Expect(properties.ContainerIP).To(Equal(net.ParseIP("169.254.1.2")))
		})

		It("should call CNI Up, passing in the bind-mounted path to the net ns", func() {
			_, err := manager.Up(42, "some-container-handle", encodedGardenProperties)
			Expect(err).NotTo(HaveOccurred())

			Expect(cniController.UpCallCount()).To(Equal(1))
			namespacePath, handle, spec := cniController.UpArgsForCall(0)
			Expect(namespacePath).To(Equal("/some/fake/path/some-container-handle"))
			Expect(handle).To(Equal("some-container-handle"))
			Expect(spec).To(Equal(encodedGardenProperties))
		})

		Context("when missing args", func() {
			It("should return a friendly error", func() {
				_, err := manager.Up(0, "some-container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("up missing pid"))

				_, err = manager.Up(42, "", encodedGardenProperties)
				Expect(err).To(MatchError("up missing container handle"))
			})
		})

		Context("when missing the network spec", func() {
			It("should be a no-op and not call CNI", func() {
				_, err := manager.Up(42, "some-container-handle", "")
				Expect(err).To(MatchError(ContainSubstring("parsing garden properties")))
			})
		})

		Context("when the encoded garden properties is an empty hash", func() {
			It("should still call CNI and the netman agent", func() {
				_, err := manager.Up(42, "some-container-handle", "{}")
				Expect(err).NotTo(HaveOccurred())

				Expect(cniController.UpCallCount()).To(Equal(1))
				Expect(mounter.IdempotentlyMountCallCount()).To(Equal(1))
			})
		})

		Context("when unmarshaling the encoded garden properties fails", func() {
			It("returns the error", func() {
				_, err := manager.Up(42, "some-container-handle", "%%%%")
				Expect(err).To(MatchError(ContainSubstring("parsing garden properties: invalid character")))
			})
		})

		Context("when the mounter fails", func() {
			It("should return the error", func() {
				mounter.IdempotentlyMountReturns(errors.New("boom"))
				_, err := manager.Up(42, "some-container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("failed mounting /proc/42/ns/net to /some/fake/path/some-container-handle: boom"))
			})
		})

		Context("when the cni Up fails", func() {
			It("should return the error", func() {
				cniController.UpReturns(nil, errors.New("bang"))
				_, err := manager.Up(42, "some-container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("cni up failed: bang"))
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

		Context("when encodedGardenProperties is empty", func() {
			It("should call CNI", func() {
				err := manager.Down("some-container-handle", "")
				Expect(err).NotTo(HaveOccurred())
				Expect(cniController.DownCallCount()).To(Equal(1))
				Expect(mounter.RemoveMountCallCount()).To(Equal(1))
			})
		})

		Context("when missing args", func() {
			It("should return a friendly error", func() {
				err := manager.Down("", "")
				Expect(err).To(MatchError("down missing container handle"))
			})
		})

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
