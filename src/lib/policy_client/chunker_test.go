package policy_client_test

import (
	"lib/models"
	"lib/policy_client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SimpleChunker", func() {
	var (
		chunker  policy_client.SimpleChunker
		policies []models.Policy
	)
	BeforeEach(func() {
		policies = []models.Policy{
			{
				Source: models.Source{
					ID: "some-app-guid",
				},
				Destination: models.Destination{
					ID:       "some-other-app-guid",
					Port:     8090,
					Protocol: "tcp",
				},
			},
			{
				Source: models.Source{
					ID: "some-app-guid-2",
				},
				Destination: models.Destination{
					ID:       "some-other-app-guid-2",
					Port:     8091,
					Protocol: "tcp",
				},
			},
			{
				Source: models.Source{
					ID: "some-app-guid-3",
				},
				Destination: models.Destination{
					ID:       "some-other-app-guid-3",
					Port:     8092,
					Protocol: "tcp",
				},
			},
		}
	})
	Context("when the policies do not chunk evenly", func() {
		BeforeEach(func() {
			chunker = policy_client.SimpleChunker{
				ChunkSize: 2,
			}
		})
		It("returns the last chunk as smaller than ChunkSize", func() {
			chunkedPolicies := chunker.Chunk(policies)
			Expect(len(chunkedPolicies)).To(Equal(2))
			Expect(chunkedPolicies[0]).To(Equal(policies[0:2]))
			Expect(chunkedPolicies[1]).To(Equal(policies[2:]))
		})
	})

	Context("when the ChunkSize is zero", func() {
		BeforeEach(func() {
			chunker = policy_client.SimpleChunker{
				ChunkSize: 0,
			}
		})
		It("chunks with a chunk size of DefaultMaxPolicies", func() {
			chunkedPolicies := chunker.Chunk(policies)
			Expect(len(chunkedPolicies)).To(Equal(1))
			Expect(chunkedPolicies[0]).To(Equal(policies))
		})
	})

})
