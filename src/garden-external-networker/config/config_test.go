package config_test

import (
	"encoding/json"
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
				file.WriteString(`{
					"cni_plugin_dir": "foo",
					"cni_config_dir": "bar",
					"bind_mount_dir": "baz",
					"state_file": "some/path",
					"start_port": 1234,
					"total_ports": 56
				}`)
				c, err := config.New(file.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(c.CniPluginDir).To(Equal("foo"))
				Expect(c.CniConfigDir).To(Equal("bar"))
				Expect(c.BindMountDir).To(Equal("baz"))
				Expect(c.StateFilePath).To(Equal("some/path"))
				Expect(c.StartPort).To(Equal(1234))
				Expect(c.TotalPorts).To(Equal(56))
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
					"cni_plugin_dir": "/some/plugin/dir",
					"cni_config_dir": "/some/config/dir",
					"bind_mount_dir": "/some/mount/dir",
					"state_file":     "/some/state/file",
					"start_port":     50000,
					"total_ports":    10000,
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
		)
	})
})
