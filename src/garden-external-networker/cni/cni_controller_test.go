package cni_test

import (
	"fmt"
	"garden-external-networker/cni"
	"garden-external-networker/fakes"

	"code.cloudfoundry.org/lager/lagertest"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CniController", func() {
	Describe("Up", func() {
		var (
			controller     cni.CNIController
			expectedResult *types.Result
			fakeCNILibrary *fakes.CNILibrary
		)

		BeforeEach(func() {
			logger := lagertest.NewTestLogger("test")
			fakeCNILibrary = &fakes.CNILibrary{}
			testConfig := &libcni.NetworkConfig{
				Network: &types.NetConf{
					CNIVersion: "some-version",
					Type:       "some-plugin",
				},
				Bytes: []byte(`{
					"cniVersion":"some-version",
					"type": "some-plugin"
				}`),
			}
			expectedResult = &types.Result{}
			fakeCNILibrary.AddNetworkReturns(expectedResult, nil)

			controller = cni.CNIController{
				Logger:    logger,
				CNIConfig: fakeCNILibrary,
				NetworkConfigs: []*libcni.NetworkConfig{
					testConfig, testConfig,
				},
			}
		})

		It("returns the result from the CNI AddNetwork call", func() {
			result, err := controller.Up("/some/namespace/path", "some-handle", map[string]string{
				"some": "properties",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeIdenticalTo(expectedResult))

			Expect(fakeCNILibrary.AddNetworkCallCount()).To(Equal(2))
			netc, runc := fakeCNILibrary.AddNetworkArgsForCall(0)
			Expect(runc.ContainerID).To(Equal("some-handle"))
			Expect(netc.Network.Type).To(Equal("some-plugin"))
		})

		Context("when the libcni.InjectConf returns an error", func() {
			It("return a meaningful error", func() {
				controller.NetworkConfigs[0].Bytes = []byte(`not valid json`)

				_, err := controller.Up("/some/namespace/path", "some-handle", map[string]string{
					"some": "properties",
				})
				Expect(err).To(MatchError(HavePrefix("adding garden properties to CNI config: unmarshal")))
			})

		})

		Context("when the AddNetwork returns an error", func() {
			It("return a meaningful error", func() {
				fakeCNILibrary.AddNetworkReturns(nil, fmt.Errorf("patato"))

				_, err := controller.Up("/some/namespace/path", "some-handle", map[string]string{
					"some": "properties",
				})
				Expect(err).To(MatchError("add network failed: patato"))
			})
		})
	})

	Describe("Down", func() {
		var (
			controller     cni.CNIController
			fakeCNILibrary *fakes.CNILibrary
		)

		BeforeEach(func() {
			logger := lagertest.NewTestLogger("test")
			fakeCNILibrary = &fakes.CNILibrary{}
			testConfig := &libcni.NetworkConfig{
				Network: &types.NetConf{
					CNIVersion: "some-version",
					Type:       "some-plugin",
				},
				Bytes: []byte(`{
					"cniVersion":"some-version",
					"type": "some-plugin"
				}`),
			}
			fakeCNILibrary.DelNetworkReturns(nil)

			controller = cni.CNIController{
				Logger:    logger,
				CNIConfig: fakeCNILibrary,
				NetworkConfigs: []*libcni.NetworkConfig{
					testConfig, testConfig,
				},
			}
		})

		It("returns no error from the CNI DeleteNetwork call", func() {
			err := controller.Down("/some/namespace/path", "some-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeCNILibrary.DelNetworkCallCount()).To(Equal(2))
			netc, runc := fakeCNILibrary.DelNetworkArgsForCall(0)
			Expect(runc.ContainerID).To(Equal("some-handle"))
			Expect(netc.Network.Type).To(Equal("some-plugin"))
		})

		Context("when the DelNetwork returns an error", func() {
			It("return a meaningful error", func() {
				fakeCNILibrary.DelNetworkReturns(fmt.Errorf("patato"))

				err := controller.Down("/some/namespace/path", "some-handle")
				Expect(err).To(MatchError("del network failed: patato"))
			})
		})
	})

})
