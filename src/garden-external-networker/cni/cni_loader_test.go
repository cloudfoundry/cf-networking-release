package cni_test

import (
	"garden-external-networker/cni"
	"io/ioutil"
	"path/filepath"

	"github.com/containernetworking/cni/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetNetworkConfigs", func() {
	var (
		cniLoader             *cni.CNILoader
		dir                   string
		err                   error
		expectedBridgeNetwork *types.NetConf
		expectedVxlanNetwork  *types.NetConf
	)

	BeforeEach(func() {
		dir, err = ioutil.TempDir("", "test-cni-dir")
		Expect(err).NotTo(HaveOccurred())

		cniLoader = &cni.CNILoader{
			PluginDir: "",
			ConfigDir: dir,
		}

		expectedBridgeNetwork = &types.NetConf{
			Name: "mynet",
			Type: "bridge",
		}
		expectedVxlanNetwork = &types.NetConf{
			Name: "mynet2",
			Type: "vxlan",
		}
	})

	Context("when the config dir does not exist", func() {
		BeforeEach(func() {
			cniLoader.ConfigDir = "/thisdoesnot/exist"
		})
		It("returns a meaningful error", func() {
			_, err := cniLoader.GetNetworkConfigs()
			Expect(err).To(MatchError(HavePrefix("error loading config:")))
		})
	})

	Context("when no config files exist in dir", func() {
		It("does not load any netconfig", func() {
			netListCfgs, err := cniLoader.GetNetworkConfigs()
			Expect(err).NotTo(HaveOccurred())
			Expect(netListCfgs).To(HaveLen(0))
		})
	})

	Context("when a valid config file exists", func() {
		BeforeEach(func() {
			err = ioutil.WriteFile(filepath.Join(dir, "foo.conf"), []byte(`{ "name": "mynet", "type": "bridge" }`), 0600)
			Expect(err).NotTo(HaveOccurred())
		})
		It("loads a single network config", func() {
			netListCfgs, err := cniLoader.GetNetworkConfigs()
			Expect(err).NotTo(HaveOccurred())
			Expect(netListCfgs).To(HaveLen(1))
			Expect(netListCfgs[0].Name).To(Equal("mynet"))
			Expect(netListCfgs[0].Plugins).To(HaveLen(1))
			Expect(*netListCfgs[0].Plugins[0].Network).To(Equal(*expectedBridgeNetwork))
		})
	})

	Context("when a valid config list files exists", func() {
		BeforeEach(func() {
			err = ioutil.WriteFile(filepath.Join(dir, "foo.conflist"), []byte(`{
				"name": "mynetlist",
				"plugins": [
			    { "name": "mynet2", "type": "vxlan" },
			    { "name" : "mynet", "type" : "bridge" }
			  ]
			}`), 0600)
			Expect(err).NotTo(HaveOccurred())
		})

		It("loads all network configs", func() {
			netListCfgs, err := cniLoader.GetNetworkConfigs()
			Expect(err).NotTo(HaveOccurred())
			Expect(netListCfgs).To(HaveLen(1))
			Expect(netListCfgs[0].Name).To(Equal("mynetlist"))
			Expect(netListCfgs[0].Plugins).To(HaveLen(2))
			Expect(*netListCfgs[0].Plugins[0].Network).To(Equal(*expectedVxlanNetwork))
			Expect(*netListCfgs[0].Plugins[1].Network).To(Equal(*expectedBridgeNetwork))
		})
	})

	Context("when multiple valid config and config list files exists", func() {
		BeforeEach(func() {
			err = ioutil.WriteFile(filepath.Join(dir, "foo.conf"), []byte(`{ "name": "mynet", "type": "bridge" }`), 0600)
			Expect(err).NotTo(HaveOccurred())

			err = ioutil.WriteFile(filepath.Join(dir, "foo.conflist"), []byte(`{ "name": "mynetlist", "plugins": [{ "name": "mynet2", "type": "vxlan" }] }`), 0600)
			Expect(err).NotTo(HaveOccurred())
		})

		It("loads all network configs", func() {
			netListCfgs, err := cniLoader.GetNetworkConfigs()
			Expect(err).NotTo(HaveOccurred())
			Expect(netListCfgs).To(HaveLen(2))
			Expect(netListCfgs[0].Name).To(Equal("mynet"))
			Expect(netListCfgs[0].Plugins).To(HaveLen(1))
			Expect(*netListCfgs[0].Plugins[0].Network).To(Equal(*expectedBridgeNetwork))
			Expect(netListCfgs[1].Name).To(Equal("mynetlist"))
			Expect(netListCfgs[1].Plugins).To(HaveLen(1))
			Expect(*netListCfgs[1].Plugins[0].Network).To(Equal(*expectedVxlanNetwork))
		})
	})
})
