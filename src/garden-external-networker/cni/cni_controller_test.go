package cni_test

import (
	"fmt"
	"garden-external-networker/cni"
	"garden-external-networker/fakes"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/020"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CniController", func() {
	var (
		controller     cni.CNIController
		fakeCNILibrary *fakes.CNILibrary
		expectedResult *types020.Result
		testConfig     *libcni.NetworkConfigList
	)

	BeforeEach(func() {
		fakeCNILibrary = &fakes.CNILibrary{}

		testConfig = &libcni.NetworkConfigList{
			Name:       "net-list-name",
			CNIVersion: "some-version",
			Plugins: []*libcni.NetworkConfig{
				{
					Network: &types.NetConf{
						CNIVersion: "some-version",
						Type:       "some-plugin",
					},
					Bytes: []byte(`{"cniVersion":"some-version", "type": "some-plugin"}`),
				},
			},
		}
		expectedResult = &types020.Result{}
		fakeCNILibrary.AddNetworkListReturns(expectedResult, nil)
		fakeCNILibrary.DelNetworkListReturns(nil)

		controller = cni.CNIController{
			CNIConfig: fakeCNILibrary,
			NetworkConfigLists: []*libcni.NetworkConfigList{
				testConfig, testConfig,
			},
		}
	})

	Describe("Up", func() {
		var (
			expectedNetConfBytes string
			metadata             map[string]interface{}
			legacyNetConf        map[string]interface{}
		)
		BeforeEach(func() {
			expectedNetConfBytes = `{
				"cniVersion":"some-version",
				"type":"some-plugin",
				"runtimeConfig": {
					"some-other": "value",
					"another": "something"
				},
				"metadata": {
					"some":"properties"
				}
			}`
			metadata = map[string]interface{}{
				"some": "properties",
			}
			legacyNetConf = map[string]interface{}{
				"some-other": "value",
				"another":    "something",
			}
		})

		It("returns the result from the CNI AddNetworkList call", func() {
			result, err := controller.Up("/some/namespace/path", "some-handle", metadata, legacyNetConf)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeIdenticalTo(expectedResult))

			Expect(fakeCNILibrary.AddNetworkListCallCount()).To(Equal(2))
			netc, runc := fakeCNILibrary.AddNetworkListArgsForCall(0)
			Expect(runc.ContainerID).To(Equal("some-handle"))
			Expect(netc.Name).To(Equal("net-list-name"))
			Expect(netc.CNIVersion).To(Equal("some-version"))
			Expect(netc.Plugins).To(HaveLen(1))
			Expect(netc.Plugins[0].Network.Type).To(Equal("some-plugin"))
			Expect(netc.Plugins[0].Bytes).To(MatchJSON(expectedNetConfBytes))
		})

		Context("when injecting the metadata fails", func() {
			It("return a meaningful error", func() {
				controller.NetworkConfigLists[0].Plugins[0].Bytes = []byte(`not valid json`)

				_, err := controller.Up("/some/namespace/path", "some-handle", metadata, legacyNetConf)
				Expect(err).To(MatchError(HavePrefix("adding extra data to CNI config: unmarshal")))
			})
		})

		Context("when the legacyNetConf is nil", func() {
			BeforeEach(func() {
				expectedNetConfBytes = `{
				"cniVersion":"some-version",
				"type":"some-plugin",
				"metadata": {
					"some":"properties"
				}
			}`
			})
			It("adds an empty runtimeConfig", func() {
				_, err := controller.Up("/some/namespace/path", "some-handle", metadata, nil)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeCNILibrary.AddNetworkListCallCount()).To(Equal(2))
				netc, _ := fakeCNILibrary.AddNetworkListArgsForCall(0)
				Expect(netc.Plugins[0].Bytes).To(MatchJSON(expectedNetConfBytes))
			})
		})

		Context("when the AddNetworkList returns an error", func() {
			It("return a meaningful error", func() {
				fakeCNILibrary.AddNetworkListReturns(nil, fmt.Errorf("patato"))

				_, err := controller.Up("/some/namespace/path", "some-handle", metadata, legacyNetConf)
				Expect(err).To(MatchError("add network list failed: patato"))
			})
		})
	})

	Describe("Down", func() {
		It("returns no error from the CNI DeleteNetwork call", func() {
			err := controller.Down("/some/namespace/path", "some-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeCNILibrary.DelNetworkListCallCount()).To(Equal(2))
			netc, runc := fakeCNILibrary.DelNetworkListArgsForCall(0)
			Expect(runc.ContainerID).To(Equal("some-handle"))
			Expect(netc.Plugins).To(HaveLen(1))
			Expect(netc.Plugins[0].Network.Type).To(Equal("some-plugin"))
		})

		Context("when the DelNetwork returns an error", func() {
			It("return a meaningful error", func() {
				fakeCNILibrary.DelNetworkListReturns(fmt.Errorf("patato"))

				err := controller.Down("/some/namespace/path", "some-handle")
				Expect(err).To(MatchError("del network failed: patato"))
			})
		})
	})
})
