package main

import (
	"fmt"
	"hash/maphash"
	"log"
	"math/rand/v2"
	"sort"
)

// Arbitrary seed
const (
	SORT_SEED1 = 1234
	SORT_SEED2 = 4321
	SORT_N     = 1024
)

// BenchSort is a sorting benchmark using the standard library facilities.
// Items
type BenchSort struct {
	array []SortItem
	r     *rand.Rand
	pcg   *rand.PCG
	h     uint64
	hseed maphash.Seed
}

// SortItem includes local and remote memory. We sort the structures using strings and integers.
// The size of the structure is not trivial to put a bit more pressure on memory.
type SortItem struct {
	firstname string
	lastname  string
	id        int
	blob      [128]byte
}

// ByFirst enforces firstname order
type ByFirst []SortItem

func (a ByFirst) Len() int           { return len(a) }
func (a ByFirst) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByFirst) Less(i, j int) bool { return a[i].firstname < a[j].firstname }

// ByLast enforces lastname order
type ByLast []SortItem

func (a ByLast) Len() int           { return len(a) }
func (a ByLast) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLast) Less(i, j int) bool { return a[i].lastname < a[j].lastname }

// ById enforces id order
type ById []SortItem

func (a ById) Len() int           { return len(a) }
func (a ById) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ById) Less(i, j int) bool { return a[i].id < a[j].id }

// NewBenchSort creates a new sorting benchmark
func NewBenchSort() *BenchSort {

	// Fill the structures with pseudo-random (but seeded) data
	pcg := rand.NewPCG(SORT_SEED1, SORT_SEED2)
	r := rand.New(pcg)
	chacha := rand.NewChaCha8([32]byte{})
	items := make([]SortItem, SORT_N)
	for i := 0; i < len(items); i++ {
		x := &(items[i])
		x.firstname = fmt.Sprintf("%016x", r.Int64())
		x.lastname = fmt.Sprintf("%016X", r.Int64())
		x.id = i
		if n, err := chacha.Read(x.blob[:]); err != nil || n != 128 {
			log.Fatal("Wrond rand read")
		}
	}

	res := &BenchSort{
		array: items,
		r:     r,
		pcg:   pcg,
		hseed: maphash.MakeSeed(),
	}

	// Calculate a hash corresponding to all the records in id order
	res.h = res.sortHash()
	return res
}

// Run the sorting benchmark
func (b *BenchSort) Run() {

	// Shuffle the slice of records
	b.pcg.Seed(SORT_SEED1, SORT_SEED2)
	b.r.Shuffle(SORT_N, func(i, j int) {
		b.array[i], b.array[j] = b.array[j], b.array[i]
	})

	// Sort in various orders
	sort.Sort(ByFirst(b.array))
	sort.Sort(ByLast(b.array))
	sort.Sort(ById(b.array))

	// We should be back in the initial order (by id), so calculate a hash and check
	if b.sortHash() != b.h {
		log.Fatal("Hash discrepancy")
	}
}

// sortHash calculate a hash code of all the records
func (b *BenchSort) sortHash() uint64 {
	var h maphash.Hash
	h.SetSeed(b.hseed)
	for i := range b.array {
		h.Write(b.array[i].blob[:])
	}
	return h.Sum64()
}
