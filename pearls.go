package main

import "log"

// Arbitrary seed
const PEARLS_N = 500

// BenchPearls is a simple implementation of Dijkstra's pearls problem.
// How to string colored beads (3 colors) so there is no two adjacent identical sequences?
type BenchPearls struct {
	buf  []byte
	iter int
}

// NewBenchPearls creates a new pearls benchmark
func NewBenchPearls() *BenchPearls {

	return &BenchPearls{
		buf:  make([]byte, 0, PEARLS_N),
		iter: PEARLS_N,
	}
}

// Run the pearls benchmark
func (b *BenchPearls) Run() {

	b.buf = b.buf[:0]
	b.iter = PEARLS_N

	b.addPearls()

	// Sanity check: the length of the result string is necessarily smaller
	// than the number of iterations
	if len(b.buf) != 451 {
		log.Fatal("Pearls problem solution mismatch")
	}
}

// addPearls add pearls up to the maximum iteration
func (b *BenchPearls) addPearls() {

	stop := b.iter < 0
	b.iter--
	if stop {
		return
	}

	for color := byte('0'); color <= '2'; color++ {
		b.buf = append(b.buf, color)
		if b.check() {
			b.addPearls()
			return
		}
		b.buf = b.buf[:len(b.buf)-1]
	}

	for {
		b.backtrack()
		if b.check() {
			break
		}
	}

	b.addPearls()
}

// check returns true when the whole sequence is valid
//go:noinline
func (b *BenchPearls) check() bool {

	max := len(b.buf)
	if max <= 1 {
		return true
	}

	for i := 2; i <= max; i += 2 {
		if !b.checkSub(max-i, max-i/2) {
			return false
		}
	}

	return true
}

// checkSub returns true when the sub-sequences ending/starting at pivot are valid
func (b *BenchPearls) checkSub(start, pivot int) bool {

	for i := 0; i < pivot-start; i++ {
		if b.buf[start+i] != b.buf[pivot+i] {
			return true
		}
	}
	return false
}

// backtrack rewinds the algorithm
func (b *BenchPearls) backtrack() {

	max := len(b.buf) - 1

	// Check last added pearl
	if b.buf[max] != '2' {

		// Change color
		b.buf[max]++

	} else {

		// Remove pearl and backtrack
		b.buf = b.buf[:max]
		b.backtrack()
	}
}
