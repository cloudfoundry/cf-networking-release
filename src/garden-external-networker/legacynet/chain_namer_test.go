package legacynet_test

import (
	. "garden-external-networker/legacynet"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ChainNamer", func() {
	var namer *ChainNamer

	BeforeEach(func() {
		namer = &ChainNamer{28}
	})

	Context("Name", func() {
		Context("when the handle is over 28 characters", func() {
			It("truncates the handle", func() {
				chainName := namer.Name("netout", "a-very-long-container-handle")
				Expect(chainName).To(Equal("netout--a-very-long-containe"))
			})
		})
	})

})
