package policy_client

import "policy-server/models"

const DefaultMaxPolicies = 100

//go:generate counterfeiter -o ../fakes/chunker.go --fake-name Chunker . Chunker
type Chunker interface {
	Chunk(allPolicies []models.Policy) [][]models.Policy
}

type SimpleChunker struct {
	ChunkSize int
}

func (c *SimpleChunker) getChunkSize() int {
	if c.ChunkSize <= 0 {
		return DefaultMaxPolicies
	}
	return c.ChunkSize
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *SimpleChunker) Chunk(allPolicies []models.Policy) [][]models.Policy {
	chunkSize := c.getChunkSize()
	chunkedPolicies := [][]models.Policy{}
	for i := 0; i < len(allPolicies); i += chunkSize {
		chunkedPolicies = append(chunkedPolicies, allPolicies[i:min(len(allPolicies), i+chunkSize)])
	}
	return chunkedPolicies
}
