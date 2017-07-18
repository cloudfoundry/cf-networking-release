package datastore_test

import (
	"errors"
	"io/ioutil"
	"os"
	"sync"

	"lib/datastore"
	libfakes "lib/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Datastore", func() {
	var (
		handle   string
		ip       string
		store    *datastore.Store
		metadata map[string]interface{}

		serializer  *libfakes.Serializer
		locker      *libfakes.Locker
		dataFile    *os.File
		versionFile *os.File
	)

	BeforeEach(func() {
		handle = "some-handle"
		ip = "192.168.0.100"
		locker = &libfakes.Locker{}
		serializer = &libfakes.Serializer{}
		metadata = map[string]interface{}{
			"AppID":         "some-appid",
			"OrgID":         "some-orgid",
			"PolicyGroupID": "some-policygroupid",
			"SpaceID":       "some-spaceid",
			"randomKey":     "randomValue",
		}

		var err error
		dataFile, err = ioutil.TempFile(os.TempDir(), "dataFile")
		Expect(err).NotTo(HaveOccurred())
		versionFile, err = ioutil.TempFile(os.TempDir(), "versionFile")
		Expect(err).NotTo(HaveOccurred())

		store = &datastore.Store{
			Serializer:      serializer,
			Locker:          locker,
			DataFilePath:    dataFile.Name(),
			VersionFilePath: versionFile.Name(),
			CacheMutex:      new(sync.RWMutex),
		}
	})

	Context("when adding an entry to store", func() {
		It("deserializes the data from the file", func() {
			err := store.Add(handle, ip, metadata)
			Expect(err).NotTo(HaveOccurred())

			Expect(locker.LockCallCount()).To(Equal(1))
			Expect(locker.UnlockCallCount()).To(Equal(1))

			Expect(serializer.DecodeAllCallCount()).To(Equal(1))
			Expect(serializer.EncodeAndOverwriteCallCount()).To(Equal(1))

			file, _ := serializer.DecodeAllArgsForCall(0)
			Expect(file.(*os.File).Name()).To(Equal(dataFile.Name()))

			_, actual := serializer.EncodeAndOverwriteArgsForCall(0)
			expected := map[string]datastore.Container{
				handle: datastore.Container{
					Handle:   handle,
					IP:       ip,
					Metadata: metadata,
				},
			}
			Expect(actual).To(Equal(expected))
		})

		It("updates the version", func() {
			err := store.Add(handle, ip, metadata)
			Expect(err).NotTo(HaveOccurred())

			versionContents, err := ioutil.ReadFile(versionFile.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(string(versionContents)).To(Equal("2"))

			err = store.Add(handle, ip, metadata)
			Expect(err).NotTo(HaveOccurred())

			versionContents, err = ioutil.ReadFile(versionFile.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(string(versionContents)).To(Equal("3"))
		})

		Context("when handle is not valid", func() {
			It("wraps and returns the error", func() {
				err := store.Add("", ip, metadata)
				Expect(err).To(MatchError("invalid handle"))
			})
		})

		Context("when input IP is not valid", func() {
			It("wraps and returns the error", func() {
				err := store.Add(handle, "invalid-ip", metadata)
				Expect(err).To(MatchError("invalid ip: invalid-ip"))
			})
		})

		Context("when the locker fails to lock", func() {
			BeforeEach(func() {
				locker.LockReturns(errors.New("potato"))
			})
			It("wraps and returns the error", func() {
				err := store.Add(handle, ip, metadata)
				Expect(err).To(MatchError("lock: potato"))
			})
		})

		Context("when the data file fails to open", func() {
			BeforeEach(func() {
				store.DataFilePath = "/some/bad/path"
			})
			It("wraps and returns the error", func() {
				err := store.Add(handle, ip, metadata)
				Expect(err).To(MatchError("open data file: open /some/bad/path: no such file or directory"))
			})
		})

		Context("when the version fails to update", func() {
			BeforeEach(func() {
				store.VersionFilePath = "/some/bad/path"
			})
			It("passes through the error", func() {
				err := store.Add(handle, ip, metadata)
				Expect(err).To(MatchError("open version file: open /some/bad/path: no such file or directory"))
			})
		})

		Context("when serializer fails to decode", func() {
			BeforeEach(func() {
				serializer.DecodeAllReturns(errors.New("potato"))
			})
			It("wraps and returns the error", func() {
				err := store.Add(handle, ip, metadata)
				Expect(err).To(MatchError("decoding file: potato"))
			})
		})

		Context("when serializer fails to encode", func() {
			BeforeEach(func() {
				serializer.EncodeAndOverwriteReturns(errors.New("potato"))
			})
			It("wraps and returns the error", func() {
				err := store.Add(handle, ip, metadata)
				Expect(err).To(MatchError("encode and overwrite: potato"))
			})
		})

	})

	Context("when deleting an entry from store", func() {
		It("deserializes the data from the file", func() {
			_, err := store.Delete(handle)
			Expect(err).NotTo(HaveOccurred())

			Expect(locker.LockCallCount()).To(Equal(1))
			Expect(locker.UnlockCallCount()).To(Equal(1))

			Expect(serializer.DecodeAllCallCount()).To(Equal(1))
			Expect(serializer.EncodeAndOverwriteCallCount()).To(Equal(1))

			file, _ := serializer.DecodeAllArgsForCall(0)
			Expect(file.(*os.File).Name()).To(Equal(dataFile.Name()))

			_, actual := serializer.EncodeAndOverwriteArgsForCall(0)
			Expect(actual).ToNot(HaveKey(handle))
		})

		It("updates the version", func() {
			_, err := store.Delete(handle)
			Expect(err).NotTo(HaveOccurred())

			versionContents, err := ioutil.ReadFile(versionFile.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(string(versionContents)).To(Equal("2"))

			_, err = store.Delete(handle)
			Expect(err).NotTo(HaveOccurred())

			versionContents, err = ioutil.ReadFile(versionFile.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(string(versionContents)).To(Equal("3"))
		})

		Context("when the data file fails to open", func() {
			BeforeEach(func() {
				store.DataFilePath = "/some/bad/path"
			})
			It("wraps and returns the error", func() {
				_, err := store.Delete(handle)
				Expect(err).To(MatchError("open data file: open /some/bad/path: no such file or directory"))
			})
		})

		Context("when the version fails to update", func() {
			BeforeEach(func() {
				store.VersionFilePath = "/some/bad/path"
			})
			It("passes the error through", func() {
				_, err := store.Delete(handle)
				Expect(err).To(MatchError("open version file: open /some/bad/path: no such file or directory"))
			})
		})

		Context("when handle is not valid", func() {
			It("wraps and returns the error", func() {
				_, err := store.Delete("")
				Expect(err).To(MatchError("invalid handle"))
			})
		})

		Context("when the locker fails to lock", func() {
			BeforeEach(func() {
				locker.LockReturns(errors.New("potato"))
			})
			It("wraps and returns the error", func() {
				_, err := store.Delete(handle)
				Expect(err).To(MatchError("lock: potato"))
			})
		})

		Context("when serializer fails to decode", func() {
			BeforeEach(func() {
				serializer.DecodeAllReturns(errors.New("potato"))
			})
			It("wraps and returns the error", func() {
				_, err := store.Delete(handle)
				Expect(err).To(MatchError("decoding file: potato"))
			})
		})

		Context("when serializer fails to encode", func() {
			BeforeEach(func() {
				serializer.EncodeAndOverwriteReturns(errors.New("potato"))
			})
			It("wraps and returns the error", func() {
				_, err := store.Delete(handle)
				Expect(err).To(MatchError("encode and overwrite: potato"))
			})
		})

	})

	Context("when reading from datastore", func() {
		It("deserializes the data from the file", func() {
			data, err := store.ReadAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(data).NotTo(BeNil())

			Expect(locker.LockCallCount()).To(Equal(1))
			Expect(locker.UnlockCallCount()).To(Equal(1))

			Expect(serializer.DecodeAllCallCount()).To(Equal(1))
			Expect(serializer.EncodeAndOverwriteCallCount()).To(Equal(0))

			file, _ := serializer.DecodeAllArgsForCall(0)
			Expect(file.(*os.File).Name()).To(Equal(dataFile.Name()))
		})

		Context("when the version has not changed", func() {
			BeforeEach(func() {
				data, err := store.ReadAll()
				Expect(err).NotTo(HaveOccurred())
				Expect(data).NotTo(BeNil())

				data, err = store.ReadAll()
				Expect(err).NotTo(HaveOccurred())
				Expect(data).NotTo(BeNil())
			})

			It("does not call lock or decode again", func() {
				Expect(locker.LockCallCount()).To(Equal(1))
				Expect(locker.UnlockCallCount()).To(Equal(1))
				Expect(serializer.DecodeAllCallCount()).To(Equal(1))
			})
		})

		Context("when the version has changed", func() {
			BeforeEach(func() {
				data, err := store.ReadAll()
				Expect(err).NotTo(HaveOccurred())
				Expect(data).NotTo(BeNil())

				err = ioutil.WriteFile(store.VersionFilePath, []byte("3"), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())

				data, err = store.ReadAll()
				Expect(err).NotTo(HaveOccurred())
				Expect(data).NotTo(BeNil())
			})

			It("calls lock and reads/decodes the data again", func() {
				Expect(locker.LockCallCount()).To(Equal(2))
				Expect(locker.UnlockCallCount()).To(Equal(2))
				Expect(serializer.DecodeAllCallCount()).To(Equal(2))
			})
		})

		Context("when getting the version fails", func() {
			BeforeEach(func() {
				store.VersionFilePath = "/some/bad/path"
			})
			It("passes the error through", func() {
				_, err := store.ReadAll()
				Expect(err).To(MatchError("open version file: open /some/bad/path: no such file or directory"))
			})
		})

		Context("when the locker fails to lock", func() {
			BeforeEach(func() {
				locker.LockReturns(errors.New("potato"))
			})
			It("wraps and returns the error", func() {
				_, err := store.ReadAll()
				Expect(err).To(MatchError("lock: potato"))
			})
		})

		Context("when the data file fails to open", func() {
			BeforeEach(func() {
				store.DataFilePath = "/some/bad/path"
			})
			It("wraps and returns the error", func() {
				_, err := store.ReadAll()
				Expect(err).To(MatchError("open data file: open /some/bad/path: no such file or directory"))
			})
		})

		Context("when serializer fails to decode", func() {
			BeforeEach(func() {
				serializer.DecodeAllReturns(errors.New("potato"))
			})
			It("wraps and returns the error", func() {
				_, err := store.ReadAll()
				Expect(err).To(MatchError("decoding file: potato"))
			})
		})
	})

	Describe("updateVersion", func() {
		Context("when the version file fails to open", func() {
			BeforeEach(func() {
				store.VersionFilePath = "/some/bad/path"
			})
			It("wraps and returns the error", func() {
				err := store.Add(handle, ip, metadata)
				Expect(err).To(MatchError("open version file: open /some/bad/path: no such file or directory"))
			})
		})

		Context("when the version file has non-version contents", func() {
			BeforeEach(func() {
				err := ioutil.WriteFile(store.VersionFilePath, []byte("foo"), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())
			})
			It("wraps and returns the error", func() {
				err := store.Add(handle, ip, metadata)
				Expect(err).To(MatchError("version file: 'foo' is not a number"))
			})
		})
	})
})
