package repository_test

import (
	"errors"
	"iptables-logger/repository"
	"lib/datastore"
	"lib/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Container", func() {
	var (
		repo      *repository.ContainerRepo
		fakeStore *fakes.Datastore
	)
	BeforeEach(func() {
		fakeStore = &fakes.Datastore{}

		repo = &repository.ContainerRepo{
			Store: fakeStore,
		}

		containers := map[string]datastore.Container{
			"handle-1": {
				Handle: "handle-1",
				IP:     "ip-1",
				Metadata: map[string]interface{}{
					"app_id":   "app-1",
					"space_id": "space-1",
					"org_id":   "org-1",
				},
			},
			"handle-2": {
				Handle:   "handle-2",
				IP:       "ip-2",
				Metadata: map[string]interface{}{},
			},
		}

		fakeStore.ReadAllReturns(containers, nil)
	})

	Describe("GetByIP", func() {
		It("looks up the container from the store", func() {
			container, err := repo.GetByIP("ip-1")
			Expect(err).NotTo(HaveOccurred())

			Expect(container).To(Equal(repository.Container{
				Handle:  "handle-1",
				AppID:   "app-1",
				SpaceID: "space-1",
				OrgID:   "org-1",
			}))
		})

		It("looks up the container from the store", func() {
			container, err := repo.GetByIP("ip-1")
			Expect(err).NotTo(HaveOccurred())

			Expect(container).To(Equal(repository.Container{
				Handle:  "handle-1",
				AppID:   "app-1",
				SpaceID: "space-1",
				OrgID:   "org-1",
			}))
		})

		Context("when unable to read from datastore", func() {
			BeforeEach(func() {
				fakeStore.ReadAllReturns(nil, errors.New("apple"))
			})

			It("returns an error", func() {
				_, err := repo.GetByIP("ip-1")
				Expect(err).To(MatchError("read all: apple"))
			})
		})

		Context("when the app id, space id and org id is invalid", func() {
			It("returns a container with those fields as empty strings", func() {
				container, err := repo.GetByIP("ip-2")
				Expect(err).NotTo(HaveOccurred())
				Expect(container.AppID).To(BeEmpty())
				Expect(container.SpaceID).To(BeEmpty())
				Expect(container.OrgID).To(BeEmpty())
			})
		})
	})
})
