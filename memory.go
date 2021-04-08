package main

import (
	"bytes"
	"math/rand"
)

// Arbitrary seed
const MEM_SEED = 12345

// BenchMemory is a benchmark exercizing the L2/L3 cache.
// A buffer is built sequentially by aggregating scattered data.
type BenchMemory struct {
	input []byte
	idx   []int64
	r     *rand.Rand
	res   bytes.Buffer
	buf   []byte
	rd    *bytes.Reader
}

// NewBenchMemory creates a new memory benchmark
func NewBenchMemory() *BenchMemory {

	// Build a large byte array
	var input []byte
	for i := 0; i < 2; i++ {
		input = append(input, jsonAirlinesB...)
		input = append(input, jsonPopulationB...)
	}

	// Fill the index with offsets of chunks of the byte array
	idx := []int64{}
	for i := 0; i < len(input); i += 1024 {
		idx = append(idx, int64(i))
	}

	// The index will be shuffled at each iteration, but in a deterministic way
	r := rand.New(rand.NewSource(MEM_SEED))
	r.Seed(MEM_SEED)

	res := &BenchMemory{
		input: input,
		idx:   idx,
		r:     r,
		buf:   make([]byte, 128),
		rd:    bytes.NewReader(input),
	}

	return res
}

// Run the memory benchmark
func (b *BenchMemory) Run() {

	// Shuffle the index
	b.r.Shuffle(len(b.idx), func(i, j int) {
		b.idx[i], b.idx[j] = b.idx[j], b.idx[i]
	})

	// Use the index to fetch and copy the first bytes of each chunk
	for _, idx := range b.idx {
		n, err := b.rd.ReadAt(b.buf, idx)
		if err == nil && n == 128 {
			b.res.Write(b.buf)
		}
	}

	b.res.Reset()
}
