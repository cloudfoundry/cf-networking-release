package controller_test

import (
	"guardian-cni-adapter/controller"
	"io/ioutil"
	"path/filepath"

	"code.cloudfoundry.org/lager/lagertest"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CNI", func() {

	var _ = Describe("GetNetworkConfigs", func() {
		var (
			cniLoader       *controller.CNILoader
			dir             string
			err             error
			expectedNetCfgs []*types.NetConf
		)

		BeforeEach(func() {
			logger := lagertest.NewTestLogger("cniLoader")

			dir, err = ioutil.TempDir("", "test-cni-dir")
			Expect(err).NotTo(HaveOccurred())

			cniLoader = &controller.CNILoader{
				PluginDir: "",
				ConfigDir: dir,
				Logger:    logger,
			}

			expectedNetCfgs = []*types.NetConf{
				{
					Name: "mynet",
					Type: "bridge",
				},
				{
					Name: "mynet2",
					Type: "vxlan",
				},
			}
		})

		Context("when the config dir does not exist", func() {
			BeforeEach(func() {
				cniLoader.ConfigDir = "/thisdoesnot/exist"
			})
			It("returns a meaningful error", func() {
				_, err := cniLoader.GetNetworkConfigs()
				Expect(err).To(MatchError(HavePrefix("error loading config: lstat /thisdoesnot/exist")))
			})
		})

		Context("when no config files exist in dir", func() {
			It("does not load any netconfig", func() {
				netCfgs, err := cniLoader.GetNetworkConfigs()
				Expect(err).NotTo(HaveOccurred())
				Expect(netCfgs).To(HaveLen(0))
			})
		})

		Context("when a valid config file exists", func() {
			BeforeEach(func() {
				err = ioutil.WriteFile(filepath.Join(dir, "foo.conf"), []byte(`{ "name": "mynet", "type": "bridge" }`), 0600)
				Expect(err).NotTo(HaveOccurred())
			})
			It("loads a single network config", func() {
				netCfgs, err := cniLoader.GetNetworkConfigs()
				Expect(err).NotTo(HaveOccurred())
				Expect(netCfgs).To(HaveLen(1))
				Expect(netCfgs[0].Network).To(Equal(expectedNetCfgs[0]))
			})
		})

		Context("when multple valid config files exists", func() {
			BeforeEach(func() {
				err = ioutil.WriteFile(filepath.Join(dir, "foo.conf"), []byte(`{ "name": "mynet", "type": "bridge" }`), 0600)
				Expect(err).NotTo(HaveOccurred())
				err = ioutil.WriteFile(filepath.Join(dir, "bar.conf"), []byte(`{ "name": "mynet2", "type": "vxlan" }`), 0600)
				Expect(err).NotTo(HaveOccurred())
			})

			It("loads all network configs", func() {
				netCfgs, err := cniLoader.GetNetworkConfigs()
				Expect(err).NotTo(HaveOccurred())
				Expect(netCfgs).To(HaveLen(2))
				Expect(netCfgs[0].Network).To(Equal(expectedNetCfgs[1]))
				Expect(netCfgs[1].Network).To(Equal(expectedNetCfgs[0]))
			})
		})
	})

	Describe("AppendNetworkSpec", func() {
		var (
			networkSpec    string
			existingConfig *libcni.NetworkConfig
		)

		BeforeEach(func() {
			networkSpec = `{"key": "value"}`
			existingConfig = &libcni.NetworkConfig{
				Network: nil,
				Bytes:   []byte(`{"something": "some-value"}`),
			}
		})

		It("inserts the garden network properties inside the 'network' field", func() {
			newNetworkSpec, err := controller.AppendNetworkSpec(existingConfig, networkSpec)
			Expect(err).NotTo(HaveOccurred())

			Expect(newNetworkSpec.Bytes).To(MatchJSON([]byte(`
			{
				"something":"some-value",
				"network": {
					"properties": {
						"key":"value"
					}
				}
			}`)))
		})

		Context("when the network spec is empty", func() {
			It("should omit the network field", func() {
				newNetworkSpec, err := controller.AppendNetworkSpec(existingConfig, "")
				Expect(err).NotTo(HaveOccurred())

				Expect(newNetworkSpec.Bytes).To(MatchJSON([]byte(`{"something":"some-value"}`)))
			})
		})

		Context("when the network spec is empty json", func() {
			It("should omit the network field", func() {
				newNetworkSpec, err := controller.AppendNetworkSpec(existingConfig, " {  }")
				Expect(err).NotTo(HaveOccurred())

				Expect(newNetworkSpec.Bytes).To(MatchJSON([]byte(`{"something":"some-value"}`)))
			})
		})

		Context("when the existingNetConfig.Bytes is malformed JSON", func() {
			It("should return an error", func() {
				existingConfig.Bytes = []byte("%%%%%%")
				_, err := controller.AppendNetworkSpec(existingConfig, networkSpec)
				Expect(err).To(MatchError(ContainSubstring("unmarshal existing network bytes")))
			})
		})

		Context("when the network spec is malformed JSON", func() {
			It("should return an error", func() {
				_, err := controller.AppendNetworkSpec(existingConfig, "%%%%%%")
				Expect(err).To(MatchError(ContainSubstring("unmarshal garden network spec")))
			})
		})
	})
})
