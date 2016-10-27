package lib_test

import (
	"cni-wrapper-plugin/fakes"
	"cni-wrapper-plugin/lib"
	"fmt"
	"net"

	"github.com/containernetworking/cni/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LoadWrapperConfig", func() {
	var input []byte
	BeforeEach(func() {
		input = []byte(`{ "datastore": "/some/path", "delegate": { "some": "info" } }`)
	})

	It("should parse it", func() {
		result, err := lib.LoadWrapperConfig(input)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(&lib.WrapperConfig{
			Datastore: "/some/path",
			Delegate: map[string]interface{}{
				"some": "info",
			},
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

	Context("when the datastore path is not set", func() {
		BeforeEach(func() {
			input = []byte(`{ "delegate": { "some": "info" } }`)
		})

		It("should return a useful error", func() {
			_, err := lib.LoadWrapperConfig(input)
			Expect(err).To(MatchError("missing datastore path"))
		})
	})
})

var _ = Describe("DelegateAdd", func() {
	var (
		input            map[string]interface{}
		pluginController *lib.PluginController
		fakeDelegator    *fakes.Delegator
		expectedResult   *types.Result
	)

	BeforeEach(func() {
		_, expectedIPNet, _ := net.ParseCIDR("1.2.3.4/32")
		expectedResult = &types.Result{
			IP4: &types.IPConfig{
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
