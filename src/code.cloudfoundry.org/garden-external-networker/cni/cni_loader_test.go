package cni_test

import (
	"bytes"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/garden-external-networker/cni"

	"github.com/containernetworking/cni/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetNetworkConfig", func() {
	var (
		cniLoader             *cni.CNILoader
		dir                   string
		err                   error
		expectedBridgeNetwork *types.NetConf
		expectedVxlanNetwork  *types.NetConf
		logger                *bytes.Buffer
	)

	BeforeEach(func() {
		dir, err = os.MkdirTemp("", "test-cni-dir")
		Expect(err).NotTo(HaveOccurred())
		logger = &bytes.Buffer{}

		cniLoader = &cni.CNILoader{
			PluginDir: "",
			ConfigDir: dir,
			Logger:    logger,
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
			_, err := cniLoader.GetNetworkConfig()
			Expect(err).To(MatchError(HavePrefix("error loading config:")))
		})
	})

	Context("when no config files exist in dir", func() {
		It("does not load any netconfig", func() {
			netListCfgs, err := cniLoader.GetNetworkConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(netListCfgs).To(BeNil())
		})
	})

	Context("when a valid config file exists", func() {
		BeforeEach(func() {
			err = os.WriteFile(filepath.Join(dir, "foo.conf"), []byte(`{ "name": "mynet", "type": "bridge" }`), 0600)
			Expect(err).NotTo(HaveOccurred())
		})

		It("loads a single network config", func() {
			netListCfg, err := cniLoader.GetNetworkConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(netListCfg.Name).To(Equal("mynet"))
			Expect(netListCfg.Plugins).To(HaveLen(1))
			Expect(*netListCfg.Plugins[0].Network).To(Equal(*expectedBridgeNetwork))
		})
	})

	Context("when a valid config list files exists", func() {
		BeforeEach(func() {
			err = os.WriteFile(filepath.Join(dir, "foo.conflist"), []byte(`{
				"name": "mynetlist",
				"plugins": [
			    { "name": "mynet2", "type": "vxlan" },
			    { "name" : "mynet", "type" : "bridge" }
			  ]
			}`), 0600)
			Expect(err).NotTo(HaveOccurred())
		})

		It("loads all network configs", func() {
			netListCfg, err := cniLoader.GetNetworkConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(netListCfg.Name).To(Equal("mynetlist"))
			Expect(netListCfg.Plugins).To(HaveLen(2))
			Expect(*netListCfg.Plugins[0].Network).To(Equal(*expectedVxlanNetwork))
			Expect(*netListCfg.Plugins[1].Network).To(Equal(*expectedBridgeNetwork))
		})
	})

	Context("when multiple valid config and config list files exists", func() {
		BeforeEach(func() {
			err = os.WriteFile(filepath.Join(dir, "aaa.conf"), []byte(`{ "name": "mynet", "type": "bridge" }`), 0600)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(filepath.Join(dir, "zzz.conf"), []byte(`{ "name": "barnet", "type": "dummy" }`), 0600)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(filepath.Join(dir, "ccc.conflist"), []byte(`{ "name": "nopelist", "plugins": [{ "name": "badnet", "type": "bridge" }] }`), 0600)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(filepath.Join(dir, "bbb.conflist"), []byte(`{ "name": "mynetlist", "plugins": [{ "name": "mynet2", "type": "vxlan" }] }`), 0600)
			Expect(err).NotTo(HaveOccurred())
		})

		It("prefers the first sorted conflist over other lists or conf files", func() {
			netListCfg, err := cniLoader.GetNetworkConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(netListCfg.Name).To(Equal("mynetlist"))
			Expect(netListCfg.Plugins).To(HaveLen(1))
			Expect(*netListCfg.Plugins[0].Network).To(Equal(*expectedVxlanNetwork))
		})

		It("logs warning for the skipped files", func() {
			_, err := cniLoader.GetNetworkConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(logger.String()).To(ContainSubstring("Only one CNI config file or conflist (chain) will be executed"))
		})
	})
})
