package cni_test

import (
	"fmt"
	"garden-external-networker/cni"
	"garden-external-networker/fakes"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/020"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	gomegaTypes "github.com/onsi/gomega/types"
)

var _ = Describe("CniController", func() {
	var (
		controller     cni.CNIController
		fakeCNILibrary *fakes.CNILibrary
		logger         *lagertest.TestLogger
		expectedResult *types020.Result
		testConfig     *libcni.NetworkConfig
	)

	HasLogDataWith := func(key string, expectedJSON string) gomegaTypes.GomegaMatcher {
		GetNetConfBytes := func(lf lager.LogFormat) string {
			if value, ok := lf.Data[key]; ok {
				return value.(string)
			}
			return "{}"
		}
		return WithTransform(GetNetConfBytes, MatchJSON(expectedJSON))

	}

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		fakeCNILibrary = &fakes.CNILibrary{}

		testConfig = &libcni.NetworkConfig{
			Network: &types.NetConf{
				CNIVersion: "some-version",
				Type:       "some-plugin",
			},
			Bytes: []byte(`{
					"cniVersion":"some-version",
					"type": "some-plugin"
				}`),
		}
		expectedResult = &types020.Result{}
		fakeCNILibrary.AddNetworkReturns(expectedResult, nil)
		fakeCNILibrary.DelNetworkReturns(nil)

		controller = cni.CNIController{
			Logger:    logger,
			CNIConfig: fakeCNILibrary,
			NetworkConfigs: []*libcni.NetworkConfig{
				testConfig, testConfig,
			},
		}
	})

	Describe("Up", func() {

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

		It("logs the network config and runtime config", func() {
			result, err := controller.Up("/some/namespace/path", "some-handle", map[string]string{
				"some": "properties",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeIdenticalTo(expectedResult))

			Expect(logger).To(gbytes.Say(`test.up-add-network-start.*type.*some-plugin`))
			Expect(logger).To(gbytes.Say(`test.up-add-network-result.*type.*some-plugin`))
			expectedNetConfBytes := `{"cniVersion":"some-version","type":"some-plugin", "metadata":{"some":"properties"}}`
			Expect(logger.Logs()).To(ContainElement(HasLogDataWith("networkConfig", expectedNetConfBytes)))

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
		It("returns no error from the CNI DeleteNetwork call", func() {
			err := controller.Down("/some/namespace/path", "some-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeCNILibrary.DelNetworkCallCount()).To(Equal(2))
			netc, runc := fakeCNILibrary.DelNetworkArgsForCall(0)
			Expect(runc.ContainerID).To(Equal("some-handle"))
			Expect(netc.Network.Type).To(Equal("some-plugin"))
		})

		It("logs the network config and runtime config", func() {
			err := controller.Down("/some/namespace/path", "some-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(logger).To(gbytes.Say(`test.down-del-network-start.*type.*some-plugin`))
			Expect(logger).To(gbytes.Say(`test.down-del-network-result.*type.*some-plugin`))
			expectedNetConfBytes := `{"cniVersion":"some-version","type":"some-plugin"}`
			Expect(logger.Logs()).To(ContainElement(HasLogDataWith("networkConfig", expectedNetConfBytes)))
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
