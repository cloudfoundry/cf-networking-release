package styles_test

import (
	"cli-plugin/styles"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Styles", func() {
	var s *styles.StyleGroup

	BeforeEach(func() {
		s = styles.NewGroup()
	})

	Describe("AddStyle", func() {
		It("wraps a string with a style tag", func() {
			Expect(s.AddStyle("foo", "bold")).To(Equal("<BOLD>foo<RESET>"))
		})

		Context("when the style is not found", func() {
			It("does nothing", func() {
				Expect(s.AddStyle("foo", "bad-tag")).To(Equal("foo"))
			})
		})
	})

	Describe("ApplyStyles", func() {
		It("converts any found tags with their registered special character", func() {
			Expect(s.ApplyStyles("<BOLD>foo<RESET>")).To(Equal("\x1B[;1mfoo\x1B[0m"))
		})
	})
})
