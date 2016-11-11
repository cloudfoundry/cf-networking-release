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

	Describe("Prefix", func() {
		It("adds a prefix to the name", func() {
			chainName := namer.Prefix("prefix", "a-string")
			Expect(chainName).To(Equal("prefix--a-string"))
		})
		Context("when the body is over 28 characters", func() {
			It("truncates the body", func() {
				chainName := namer.Prefix("netout", "a-very-long-container-handle")
				Expect(chainName).To(Equal("netout--a-very-long-containe"))
			})
		})
	})

	Describe("Postfix", func() {
		It("adds a suffix to the body", func() {
			chainName, err := namer.Postfix("a-string", "suffix")
			Expect(err).NotTo(HaveOccurred())
			Expect(chainName).To(Equal("a-string--suffix"))
		})
		Context("when the body is over 28 characters", func() {
			It("truncates the body", func() {
				chainName, err := namer.Postfix("a-veryverylongstringofcharacters", "log")
				Expect(err).NotTo(HaveOccurred())
				Expect(chainName).To(Equal("a-veryverylongstringofc--log"))
			})
		})
	})
	Context("when the suffix length is too long", func() {
		It("returns an error", func() {
			_, err := namer.Postfix("some-string", "a-veryverylongstringofcharacters")
			Expect(err).To(MatchError("suffix too long, string could not be truncated to max length"))
		})
	})
})
