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
	sout   []string
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
		{Key: "user:1", Val: "Jane"},
		{Key: "user:2", Val: "Andy"},
		{Key: "user:3", Val: "Steve"},
		{Key: "user:4", Val: "Andrea"},
		{Key: "user:5", Val: "Janet"},
		{Key: "user:6", Val: "Andy"},
		{Key: "user:7", Val: "Didier"},
		{Key: "user:8", Val: "Didier"},
		{Key: "user:9", Val: "Stephane"},
		{Key: "user:10", Val: "Dirk"},
		{Key: "user:11", Val: "Dirk"},
		{Key: "user:12", Val: "Andy"},
		{Key: "user:13", Val: "Bidule"},
		{Key: "user:14", Val: "Dan"},
		{Key: "user:15", Val: "Anselme"},
		{Key: "user:16", Val: "John"},
		{Key: "person:1", Val: "Bob"},
		{Key: "person:2", Val: "Dan"},
		{Key: "employee:1", Val: "Peter"},
		{Key: "employee:2", Val: "Homer"},
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

	return &BenchBtree{
		users:  users,
		others: others,
		sout:   make([]string, 0, 2*len(users)),
	}
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
	for i := 0; i < 64; i++ {
		for _, user := range b.users {
			keys.Get(user)
			vals.Get(user)
		}
	}

	// Build a list of values by forward iteration
	keys.Ascend(nil, func(item interface{}) bool {
		b.sout = append(b.sout, item.(*Item).Val)
		return true
	})

	// Complete with a list of keys by forward iteration
	vals.Ascend(nil, func(item interface{}) bool {
		b.sout = append(b.sout, item.(*Item).Key)
		return true
	})

	// Sanity check
	if len(b.sout) != 2*len(b.users) {
		log.Fatal("btree: wrong number of items")
	}
	b.sout = b.sout[:0]
}

// test2 builds a bigger btree with a map, and compare items between them, before iterating on the btree
func (b *BenchBtree) test2() {

	keys := btree.New(byKeys)

	// Build the btree and the map
	m := make(map[string]*Item, 128)
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
