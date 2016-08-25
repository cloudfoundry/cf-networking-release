package config_test

import (
	"fmt"
	"garden-external-networker/config"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("New", func() {
		var (
			file *os.File
			err  error
		)

		BeforeEach(func() {
			file, err = ioutil.TempFile(os.TempDir(), "config-")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when config file is valid", func() {
			It("returns the config", func() {
				file.WriteString(`{"cni_plugin_dir": "foo", "cni_config_dir": "bar", "bind_mount_dir": "baz", "overlay_network": "10.255.0.0./16"}`)
				c, err := config.New(file.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(c.CniPluginDir).To(Equal("foo"))
				Expect(c.CniConfigDir).To(Equal("bar"))
				Expect(c.BindMountDir).To(Equal("baz"))
			})
		})

		Context("when config file path does not exist", func() {
			It("returns the error", func() {
				_, err := config.New("not-exists")
				Expect(err).To(MatchError(ContainSubstring("file does not exist:")))
			})
		})

		Context("when config file path is blank", func() {
			It("returns the error", func() {
				_, err := config.New("")
				Expect(err).To(MatchError(ContainSubstring("file does not exist:")))
			})
		})

		Context("when config file is bad format", func() {
			It("returns the error", func() {
				file.WriteString("bad-format")
				_, err = config.New(file.Name())
				Expect(err).To(MatchError(ContainSubstring("parsing config")))
			})
		})

		Context("when config file contents blank", func() {
			It("returns the error", func() {
				_, err = config.New(file.Name())
				Expect(err).To(MatchError(ContainSubstring("parsing config")))
			})
		})

		DescribeTable("when config file is missing a member",
			func(missingFlag, cpd, ccd, bmd, on string) {
				file.WriteString(fmt.Sprintf(`
				{
					"cni_plugin_dir": "%s",
					"cni_config_dir": "%s",
					"bind_mount_dir": "%s",
					"overlay_network": "%s"
				}`, cpd, ccd, bmd, on))
				_, err = config.New(file.Name())
				Expect(err).To(MatchError(fmt.Sprintf("missing required config '%s'", missingFlag)))
			},
			Entry("missing cni plugin dir", "cni_plugin_dir", "", "bar", "baz", "10.255.0.0/16"),
			Entry("missing cni config dir", "cni_config_dir", "foo", "", "baz", "10.255.0.0/16"),
			Entry("missing bind mount dir", "bind_mount_dir", "foo", "bar", "", "10.255.0.0/16"),
			Entry("missing overlay network", "overlay_network", "foo", "bar", "baz", ""),
		)
	})
})
