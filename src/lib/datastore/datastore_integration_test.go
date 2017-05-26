package datastore_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync/atomic"

	"code.cloudfoundry.org/cf-networking-helpers/testsupport"

	"lib/datastore"
	"lib/filelock"
	"lib/serial"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Datastore Lifecycle", func() {
	var (
		handle   string
		ip       string
		store    *datastore.Store
		metadata map[string]interface{}
		filepath string
	)

	BeforeEach(func() {
		handle = "some-handle"
		ip = "192.168.0.100"
		metadata = map[string]interface{}{
			"AppID":         "some-appid",
			"OrgID":         "some-orgid",
			"PolicyGroupID": "some-policygroupid",
			"SpaceID":       "some-spaceid",
			"randomKey":     "randomValue",
		}

		file, err := ioutil.TempFile("", "")
		Expect(err).NotTo(HaveOccurred())
		filepath = file.Name()

		locker := &filelock.Locker{
			Path: filepath,
		}
		serializer := &serial.Serial{}

		store = &datastore.Store{
			Serializer: serializer,
			Locker:     locker,
		}
	})

	AfterEach(func() {
		os.Remove(filepath)
	})

	Context("when empty", func() {
		It("returns an empty map", func() {
			data, err := store.ReadAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(data)).To(Equal(0))
		})
	})

	Context("when adding", func() {
		It("can add entry to datastore", func() {
			By("adding an entry to store")
			err := store.Add(handle, ip, metadata)
			Expect(err).NotTo(HaveOccurred())

			By("verify entry is in store")
			data, err := store.ReadAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(data).Should(HaveKey(handle))

			Expect(data[handle].IP).To(Equal(ip))
			for k, v := range metadata {
				Expect(data[handle].Metadata).Should(HaveKeyWithValue(k, v))
			}
		})

		It("can add multiple entries to datastore", func() {
			total := 250
			By("adding an entries to store")
			for i := 0; i < total; i++ {
				id := fmt.Sprintf("%s-%d", handle, i)
				err := store.Add(id, ip, metadata)
				Expect(err).NotTo(HaveOccurred())
			}

			By("verify entries are in store")
			data, err := store.ReadAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(data).Should(HaveLen(total))
		})
	})

	Context("when removing", func() {
		It("can add entry and remove an entry from datastore", func() {
			By("adding an entry to store")
			err := store.Add(handle, ip, metadata)
			Expect(err).NotTo(HaveOccurred())

			By("verify entry is in store")
			data, err := store.ReadAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(data).Should(HaveLen(1))

			By("removing entry from store")
			deleted, err := store.Delete(handle)
			Expect(err).NotTo(HaveOccurred())
			Expect(deleted.Handle).To(Equal(handle))
			Expect(deleted.IP).To(Equal(ip))
			Expect(deleted.Metadata).To(Equal(metadata))

			By("verify entry no longer in store")
			data, err = store.ReadAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(data).Should(BeEmpty())
		})

		It("can remove multiple entries to datastore", func() {
			total := 250
			By("adding an entries to store")
			for i := 0; i < total; i++ {
				id := fmt.Sprintf("%s-%d", handle, i)
				err := store.Add(id, ip, metadata)
				Expect(err).NotTo(HaveOccurred())
			}

			By("verify entries are in store")
			data, err := store.ReadAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(data).Should(HaveLen(total))

			By("removing entries from store")
			for i := 0; i < total; i++ {
				id := fmt.Sprintf("%s-%d", handle, i)
				deleted, err := store.Delete(id)
				Expect(deleted.Handle).To(Equal(id))
				Expect(err).NotTo(HaveOccurred())
			}

			By("verify store is empty")
			data, err = store.ReadAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(data).Should(BeEmpty())
		})
	})

	Context("when adding and deleting concurrently", func() {
		It("remains consistent", func() {

			containerHandles := []interface{}{}
			total := 250
			for i := 0; i < total; i++ {
				id := fmt.Sprintf("%s-%d", handle, i)
				containerHandles = append(containerHandles, id)
			}

			parallelRunner := &testsupport.ParallelRunner{
				NumWorkers: 50,
			}
			toDelete := make(chan (interface{}), total)
			toRead := make(chan (interface{}), total)

			go func() {
				parallelRunner.RunOnSlice(containerHandles, func(containerHandle interface{}) {
					p := containerHandle.(string)
					func(id string) {
						err := store.Add(id, ip, metadata)
						Expect(err).NotTo(HaveOccurred())
					}(p)
					toRead <- p
				})
				close(toRead)
			}()

			go func() {
				parallelRunner.RunOnChannel(toRead, func(containerHandle interface{}) {
					p := containerHandle.(string)
					func(id string) {
						contents, err := store.ReadAll()
						Expect(err).NotTo(HaveOccurred())
						Expect(contents).To(HaveKey(p))
					}(p)
					toDelete <- p
				})
				close(toDelete)
			}()

			var nDeleted int32
			parallelRunner.RunOnChannel(toDelete, func(containerHandle interface{}) {
				p := containerHandle.(string)
				func(id string) {
					_, err := store.Delete(id)
					Expect(err).NotTo(HaveOccurred())
				}(p)
				atomic.AddInt32(&nDeleted, 1)
			})
			Expect(nDeleted).To(Equal(int32(total)))

			By("adding an entries to store")
			data, err := store.ReadAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(data).Should(HaveLen(0))

		})
	})
})
