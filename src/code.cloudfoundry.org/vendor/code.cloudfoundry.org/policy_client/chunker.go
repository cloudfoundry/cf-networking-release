package policy_client

const DefaultMaxPolicies = 100

//go:generate counterfeiter -o ./fakes/chunker.go --fake-name Chunker . Chunker
type Chunker interface {
	Chunk(allPolicies []PolicyV0) [][]PolicyV0
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

func (c *SimpleChunker) Chunk(allPolicies []PolicyV0) [][]PolicyV0 {
	chunkSize := c.getChunkSize()
	chunkedPolicies := [][]PolicyV0{}
	for i := 0; i < len(allPolicies); i += chunkSize {
		chunkedPolicies = append(chunkedPolicies, allPolicies[i:min(len(allPolicies), i+chunkSize)])
	}
	return chunkedPolicies
}
