package main

import (
	"bytes"
	"math/rand/v2"
)

// Arbitrary seed
const (
	MEM_SEED1 = 12345
	MEM_SEED2 = 54321
	MEM_BLOCK = 13
)

// BenchMemory is a benchmark exercizing the L2/L3 cache.
// A buffer is built sequentially by aggregating scattered data.
// We avoid using memmove which may be optimized differently depending on the CPU.
type BenchMemory struct {
	input []byte
	idx   []int
	r     *rand.Rand
	res   bytes.Buffer
	buf   []byte
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
	idx := []int{}
	// Avoid to align memory blocks, ensure MEM_BLOCK bytes can be read from the index
	for i := 0; i < len(input)-MEM_BLOCK; i += 1023 {
		idx = append(idx, i)
	}

	// The index will be shuffled at each iteration, but in a deterministic way
	r := rand.New(rand.NewPCG(MEM_SEED1, MEM_SEED2))

	res := &BenchMemory{
		input: input,
		idx:   idx,
		r:     r,
		buf:   make([]byte, MEM_BLOCK), // Not a power of 2
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
		b.readBlock(idx)
		b.res.Write(b.buf)
	}

	b.res.Reset()
}

//go:noinline
func (b *BenchMemory) readBlock(idx int) {
	// Do not use optimized memmove mechanisms - simple loop is preferred
	for i := range b.buf {
		b.buf[i] = b.input[idx+i]
	}
}
