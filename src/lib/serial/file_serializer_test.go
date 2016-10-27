package serial_test

import (
	"errors"
	"io/ioutil"
	"lib/fakes"
	"lib/serial"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("FileSerializer", func() {
	var serializer *serial.Serial
	BeforeEach(func() {
		serializer = &serial.Serial{}
	})

	Describe("DecodeAll", func() {
		var file *strings.Reader
		var outData struct{ Some string }

		BeforeEach(func() {
			file = strings.NewReader(`{ "some": "data" }`)
		})

		It("decodes the file as JSON", func() {
			Expect(serializer.DecodeAll(file, &outData)).To(Succeed())
			Expect(outData.Some).To(Equal("data"))
		})

		Context("when the read cursor is not at the start of the file", func() {
			BeforeEach(func() {
				file.ReadByte()
			})

			It("still decodes the entire file contents", func() {
				Expect(serializer.DecodeAll(file, &outData)).To(Succeed())
				Expect(outData.Some).To(Equal("data"))
			})
		})

		Context("when the file is empty", func() {
			BeforeEach(func() {
				file = strings.NewReader("")
			})
			It("succeeds", func() {
				Expect(serializer.DecodeAll(file, &outData)).To(Succeed())
			})
		})

		Context("when seek fails", func() {
			var file *fakes.OverwriteableFile
			BeforeEach(func() {
				file = &fakes.OverwriteableFile{}
				file.SeekReturns(0, errors.New("banana"))
			})
			It("returns the error", func() {
				err := serializer.DecodeAll(file, &outData)
				Expect(err).To(MatchError("banana"))
			})
		})

		Context("when the json decode fails", func() {
			BeforeEach(func() {
				file = strings.NewReader("{{{")
			})
			It("returns the error", func() {
				err := serializer.DecodeAll(file, &outData)
				Expect(err).To(MatchError(ContainSubstring("invalid character")))
			})
		})
	})

	Describe("EncodeAndOverwrite", func() {
		var file *os.File

		BeforeEach(func() {
			var err error
			file, err = ioutil.TempFile("", "some-file.json")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(file.Close()).To(Succeed())
			Expect(os.RemoveAll(file.Name())).To(Succeed())
		})

		It("encodes the data", func() {
			outData := map[string]string{"some": "data"}
			Expect(serializer.EncodeAndOverwrite(file, outData)).To(Succeed())

			fileBytes, err := ioutil.ReadFile(file.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(fileBytes).To(MatchJSON(`{"some":"data"}`))
		})

		Context("when there is already data in the file", func() {
			BeforeEach(func() {
				file.WriteString("some old data much longer than the new data")
			})

			It("overwrites the old data with the new data", func() {
				outData := map[string]string{"some": "new data"}
				Expect(serializer.EncodeAndOverwrite(file, outData)).To(Succeed())

				fileBytes, err := ioutil.ReadFile(file.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(fileBytes).To(MatchJSON(`{"some":"new data"}`))
			})
		})

		Context("when file seek fails", func() {
			var file *fakes.OverwriteableFile
			BeforeEach(func() {
				file = &fakes.OverwriteableFile{}
				file.SeekReturns(0, errors.New("banana"))
			})
			It("returns the error", func() {
				outData := map[string]string{"some": "data"}
				Expect(serializer.EncodeAndOverwrite(file, outData)).To(MatchError("banana"))
			})
		})

		Context("when file truncate fails", func() {
			var file *fakes.OverwriteableFile
			BeforeEach(func() {
				file = &fakes.OverwriteableFile{}
				file.TruncateReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				outData := map[string]string{"some": "data"}
				Expect(serializer.EncodeAndOverwrite(file, outData)).To(MatchError("banana"))
			})
		})
	})
})
