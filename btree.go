package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/tidwall/btree"
)

// BenchBtree is a btree benchmark
type BenchBtree struct {
	users  []*Item
	others []*Item
}

// Item is just a simple key/value structure
type Item struct {
	Key, Val string
}

// byKeys is a comparison function that compares item keys
func byKeys(a, b interface{}) bool {
	i1, i2 := a.(*Item), b.(*Item)
	return i1.Key < i2.Key
}

// byVals is a comparison function that compares item values
func byVals(a, b interface{}) bool {
	i1, i2 := a.(*Item), b.(*Item)
	if i1.Val < i2.Val {
		return true
	}
	if i1.Val > i2.Val {
		return false
	}
	return byKeys(a, b)
}

// NewBenchTree allocates a new benchmark object
func NewBenchBtree() *BenchBtree {

	// Random data
	users := []*Item{
		&Item{Key: "user:1", Val: "Jane"},
		&Item{Key: "user:2", Val: "Andy"},
		&Item{Key: "user:3", Val: "Steve"},
		&Item{Key: "user:4", Val: "Andrea"},
		&Item{Key: "user:5", Val: "Janet"},
		&Item{Key: "user:6", Val: "Andy"},
		&Item{Key: "user:7", Val: "Didier"},
		&Item{Key: "user:8", Val: "Didier"},
		&Item{Key: "user:9", Val: "Stephane"},
		&Item{Key: "user:10", Val: "Dirk"},
		&Item{Key: "user:11", Val: "Dirk"},
		&Item{Key: "user:12", Val: "Andy"},
		&Item{Key: "user:13", Val: "Bidule"},
		&Item{Key: "user:14", Val: "Dan"},
		&Item{Key: "user:15", Val: "Anselme"},
		&Item{Key: "user:16", Val: "Johm"},
	}

	// Build some more items based on the JSON data
	others := []*Item{}
	s := jsonAirlines
	for i := 0; i < 1024; i++ {
		i1 := strings.IndexByte(s, '"')
		s = s[i1+1:]
		i2 := strings.IndexByte(s, '"')
		others = append(others, &Item{s[:i2], fmt.Sprintf("%04d", i)})
		s = s[i2+1:]
	}

	return &BenchBtree{users: users, others: others}
}

// Run executes a benchmark step
func (b *BenchBtree) Run() {
	b.test1()
	b.test2()
}

// test1 builds 2 small btrees, retrieves items from them, and iterate on them
func (b *BenchBtree) test1() {

	// Build the 2 btrees
	keys := btree.New(byKeys)
	vals := btree.New(byVals)

	for _, user := range b.users {
		keys.Set(user)
		vals.Set(user)
	}

	// Retrieve all values from them
	for i := 0; i < 16; i++ {
		for _, user := range b.users {
			keys.Get(user)
			vals.Get(user)
		}
	}

	s := make([]string, 0, 2*len(b.users))

	// Build a list of values by forward iteration
	keys.Ascend(nil, func(item interface{}) bool {
		s = append(s, item.(*Item).Val)
		return true
	})

	// Complete with a list of keys by forward iteration
	vals.Ascend(nil, func(item interface{}) bool {
		s = append(s, item.(*Item).Key)
		return true
	})
}

// test2 builds a bigger btree with a map, and compare items between them, before iterating on the btree
func (b *BenchBtree) test2() {

	keys := btree.New(byKeys)

	// Build the btree and the map
	m := map[string]*Item{}
	for _, other := range b.others {
		keys.Set(other)
		m[other.Key] = other
	}

	// Retrieve all items by keys, and compare the associated values
	for _, other := range b.others {
		res := keys.Get(other)
		if res == nil {
			log.Fatal("Key not found")
		}
		if strings.Compare(m[other.Key].Val, res.(*Item).Val) != 0 {
			log.Fatal("Value discrepancy")
		}
	}

	// Forward iteration on the btree to count items
	n := 0
	keys.Ascend(nil, func(item interface{}) bool {
		n++
		return true
	})
	if n != keys.Len() {
		log.Fatal("Length discrepancy")
	}
}
