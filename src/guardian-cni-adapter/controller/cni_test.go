package controller_test

import (
	"guardian-cni-adapter/controller"

	"github.com/containernetworking/cni/libcni"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CNI", func() {
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
