package config_test

import (
	"encoding/json"
	"fmt"
	"os"

	"code.cloudfoundry.org/garden-external-networker/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("New", func() {
		var (
			file *os.File
			err  error
		)

		BeforeEach(func() {
			file, err = os.CreateTemp(os.TempDir(), "config-")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when config file is valid", func() {
			It("returns the config", func() {
				file.WriteString(`{
					"cni_plugin_dir": "foo",
					"cni_config_dir": "bar",
					"bind_mount_dir": "baz",
					"state_file": "some/path",
					"start_port": 1234,
					"total_ports": 56,
					"log_prefix": "prefix",
					"iptables_lock_file": "some-lock-file",
					"proxy_redirect_cidr": "some-cidr",
					"enable_ingress_proxy_redirect": true,
					"proxy_port": 1111,
					"proxy_uid": 1,
					"search_domains": [
						"pivotal.io",
						"foo.bar",
						"baz.me"
					]
				}`)
				c, err := config.New(file.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(c.CniPluginDir).To(Equal("foo"))
				Expect(c.CniConfigDir).To(Equal("bar"))
				Expect(c.BindMountDir).To(Equal("baz"))
				Expect(c.StateFilePath).To(Equal("some/path"))
				Expect(c.StartPort).To(Equal(uint32(1234)))
				Expect(c.TotalPorts).To(Equal(uint32(56)))
				Expect(c.LogPrefix).To(Equal("prefix"))
				Expect(c.SearchDomains).Should(ConsistOf("pivotal.io", "foo.bar", "baz.me"))
				Expect(c.IPTablesLockFile).To(Equal("some-lock-file"))
				Expect(c.ProxyRedirectCIDR).To(Equal("some-cidr"))
				Expect(c.EnableIngressProxyRedirect).To(BeTrue())
				Expect(c.ProxyPort).To(Equal(1111))
				Expect(*c.ProxyUID).To(Equal(1))
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
			func(missingFlag string) {
				allData := map[string]interface{}{
					"cni_plugin_dir":      "/some/plugin/dir",
					"cni_config_dir":      "/some/config/dir",
					"bind_mount_dir":      "/some/mount/dir",
					"state_file":          "/some/state/file",
					"start_port":          50000,
					"total_ports":         10000,
					"log_prefix":          "prefix",
					"iptables_lock_file":  "some-lock-file",
					"proxy_redirect_cidr": "some-cidr", // optional
					"proxy_port":          1111,
					"proxy_uid":           1,
				}
				delete(allData, missingFlag)
				Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())

				_, err = config.New(file.Name())
				Expect(err).To(MatchError(fmt.Sprintf("missing required config '%s'", missingFlag)))
			},
			Entry("missing cni plugin dir", "cni_plugin_dir"),
			Entry("missing cni config dir", "cni_config_dir"),
			Entry("missing bind mount dir", "bind_mount_dir"),
			Entry("missing state file", "state_file"),
			Entry("missing start port", "start_port"),
			Entry("missing total ports", "total_ports"),
			Entry("missing log prefix", "log_prefix"),
			Entry("missing iptables_lock_file", "iptables_lock_file"),
			Entry("missing proxy_port", "proxy_port"),
			Entry("missing proxy_uid", "proxy_uid"),
		)
	})
})
