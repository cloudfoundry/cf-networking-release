package policy_client

import "policy-server/api/api_0_0_0"

const DefaultMaxPolicies = 100

//go:generate counterfeiter -o ../fakes/chunker.go --fake-name Chunker . Chunker
type Chunker interface {
	Chunk(allPolicies []api_0_0_0.Policy) [][]api_0_0_0.Policy
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

func (c *SimpleChunker) Chunk(allPolicies []api_0_0_0.Policy) [][]api_0_0_0.Policy {
	chunkSize := c.getChunkSize()
	chunkedPolicies := [][]api_0_0_0.Policy{}
	for i := 0; i < len(allPolicies); i += chunkSize {
		chunkedPolicies = append(chunkedPolicies, allPolicies[i:min(len(allPolicies), i+chunkSize)])
	}
	return chunkedPolicies
}
