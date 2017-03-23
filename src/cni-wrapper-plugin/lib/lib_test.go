package lib_test

import (
	"cni-wrapper-plugin/fakes"
	"cni-wrapper-plugin/lib"
	"encoding/json"
	"fmt"
	"net"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/020"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("LoadWrapperConfig", func() {
	var input []byte
	BeforeEach(func() {
		input = []byte(`{
			"datastore": "/some/path",
			"iptables_lock_file": "/some/other/path",
			"overlay_network": "10.255.0.0/16",
			"health_check_url": "http://127.0.0.1:10007",
			"instance_address": "10.244.20.1",
			"iptables_asg_logging": true,
			"delegate": {
				"some": "info"
			}
		}`)
	})

	It("should parse it", func() {
		result, err := lib.LoadWrapperConfig(input)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(&lib.WrapperConfig{
			Datastore:          "/some/path",
			IPTablesLockFile:   "/some/other/path",
			OverlayNetwork:     "10.255.0.0/16",
			InstanceAddress:    "10.244.20.1",
			IPTablesASGLogging: true,
			Delegate: map[string]interface{}{
				"some": "info",
			},
			HealthCheckURL: "http://127.0.0.1:10007",
		}))
	})

	Context("When the stdin is not a valid json", func() {
		BeforeEach(func() {
			input = []byte("}{")
		})

		It("should return a useful error", func() {
			_, err := lib.LoadWrapperConfig(input)
			Expect(err).To(MatchError(HavePrefix("loading wrapper config: ")))
		})
	})

	DescribeTable("missing required field", func(field, errMessage string) {
		var config map[string]interface{}
		Expect(json.Unmarshal(input, &config)).To(Succeed())
		delete(config, field)

		var err error
		input, err = json.Marshal(config)
		Expect(err).NotTo(HaveOccurred())
		_, err = lib.LoadWrapperConfig(input)
		Expect(err).To(MatchError(errMessage))
	},
		Entry("datastore", "datastore", "missing datastore path"),
		Entry("ip tables lock file", "iptables_lock_file", "missing iptables lock file path"),
		Entry("overlay network", "overlay_network", "missing overlay network"),
		Entry("health check url", "health_check_url", "missing health check url"),
		Entry("instance address", "instance_address", "missing instance address"),
	)
})

var _ = Describe("DelegateAdd", func() {
	var (
		input            map[string]interface{}
		pluginController *lib.PluginController
		fakeDelegator    *fakes.Delegator
		expectedResult   types.Result
	)

	BeforeEach(func() {
		_, expectedIPNet, _ := net.ParseCIDR("1.2.3.4/32")
		expectedResult = &types020.Result{
			IP4: &types020.IPConfig{
				IP: *expectedIPNet,
			},
		}
		fakeDelegator = &fakes.Delegator{}
		fakeDelegator.DelegateAddReturns(expectedResult, nil)
		pluginController = &lib.PluginController{
			Delegator: fakeDelegator,
		}

		input = map[string]interface{}{
			"type": "something",
		}
	})

	It("should call the plugin specified by the type", func() {
		result, err := pluginController.DelegateAdd(input)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(expectedResult))
	})

	Context("when the input cannot be serialized into json", func() {
		BeforeEach(func() {
			input = map[string]interface{}{
				"bad-data": make(chan bool),
			}
		})

		It("should return a useful error", func() {
			_, err := pluginController.DelegateAdd(input)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(HavePrefix("serializing delegate netconf:")))
		})
	})

	Context("when the delegator returns an error", func() {
		BeforeEach(func() {
			fakeDelegator.DelegateAddReturns(nil, fmt.Errorf("patato"))
		})

		It("should return a useful error", func() {
			_, err := pluginController.DelegateAdd(input)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("patato"))
		})
	})

	Context("when the input type is missing", func() {
		BeforeEach(func() {
			input = map[string]interface{}{
				"notype": "shoudbemissing",
			}

		})

		It("should return a useful error", func() {
			_, err := pluginController.DelegateAdd(input)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("delegate config is missing type"))
		})
	})
})

var _ = Describe("DelegateDel", func() {
	var (
		input            map[string]interface{}
		pluginController *lib.PluginController
		fakeDelegator    *fakes.Delegator
	)

	BeforeEach(func() {
		fakeDelegator = &fakes.Delegator{}
		fakeDelegator.DelegateDelReturns(nil)
		pluginController = &lib.PluginController{
			Delegator: fakeDelegator,
		}

		input = map[string]interface{}{
			"type": "something",
		}
	})

	It("should call the plugin specified by the type", func() {
		err := pluginController.DelegateDel(input)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when the input cannot be serialized into json", func() {
		BeforeEach(func() {
			input = map[string]interface{}{
				"bad-data": make(chan bool),
			}
		})

		It("should return a useful error", func() {
			err := pluginController.DelegateDel(input)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(HavePrefix("serializing delegate netconf:")))
		})
	})

	Context("when the delegator returns an error", func() {
		BeforeEach(func() {
			fakeDelegator.DelegateDelReturns(fmt.Errorf("patato"))
		})

		It("should return a useful error", func() {
			err := pluginController.DelegateDel(input)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("patato"))
		})
	})

	Context("when the input type is missing", func() {
		BeforeEach(func() {
			input = map[string]interface{}{
				"notype": "shoudbemissing",
			}
		})

		It("should return a useful error", func() {
			err := pluginController.DelegateDel(input)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("delegate config is missing type"))
		})
	})
})
