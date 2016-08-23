package cni_test

import (
	"garden-external-networker/cni"
	"io/ioutil"
	"path/filepath"

	"code.cloudfoundry.org/lager/lagertest"

	"github.com/containernetworking/cni/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CNI", func() {

	var _ = Describe("GetNetworkConfigs", func() {
		var (
			cniLoader       *cni.CNILoader
			dir             string
			err             error
			expectedNetCfgs []*types.NetConf
		)

		BeforeEach(func() {
			logger := lagertest.NewTestLogger("cniLoader")

			dir, err = ioutil.TempDir("", "test-cni-dir")
			Expect(err).NotTo(HaveOccurred())

			cniLoader = &cni.CNILoader{
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
})
