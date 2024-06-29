package main

import (
	"bytes"
	"fmt"
	"math"
)

// BenchHaversine is a benchmark to find the optimal journey involving all cities
// (i.e. traveling salesman problem) by trying permutations of intermediate points
// and calculation of the haversine spherical distance.
type BenchHaversine struct {
	cities  []City
	idx     []int
	journey []int
	buf     bytes.Buffer
}

// City represents a city and its coordinates
type City struct {
	name     string
	lat, lng float64
}

// NewBenchHaversine allocates a new benchmark object
func NewBenchHaversine() *BenchHaversine {

	// Random cities - the journey starts and end in Paris
	cities := []City{
		{"Paris", 48.8567, 2.3522},
		{"Portland", 45.5371, -122.65},
		{"London", 51.5072, -0.1275},
		{"New-York", 40.6943, -73.9249},
		{"Seattle", 47.6211, -122.3244},
		{"Amsterdam", 52.3728, 4.8936},
		{"Dallas", 32.7935, -96.7667},
		{"Paris", 48.8567, 2.3522},
	}

	ret := &BenchHaversine{
		cities:  cities,
		idx:     make([]int, len(cities)),
		journey: make([]int, len(cities)),
	}
	return ret
}

// Run does find the optimal journey involving all cities (classical travelling salesman problem)
func (b *BenchHaversine) Run() {

	b.buf.Reset()
	best := math.MaxFloat64

	// This function will be called for each permutation
	yield := func() {

		sum := 0.0

		// Calculate the length of the journey by summing the distances
		for i := range b.idx[:len(b.idx)-1] {
			sum += b.distance(b.idx[i], b.idx[i+1])
		}

		// Keep the best one
		if sum < best {
			best = sum
			copy(b.journey, b.idx)
		}
	}

	// Initialize index array with default order
	for i := 0; i < len(b.cities); i++ {
		b.idx[i] = i
		b.journey[i] = 0
	}

	// Launch Heap's algorithm to generate all permutations
	subset := b.idx[1 : len(b.idx)-1]
	b.heap(len(subset), subset, yield)

	// Format result
	for i := range b.journey {
		fmt.Fprintf(&b.buf, "%s ", b.cities[b.journey[i]].name)
	}
	fmt.Fprintf(&b.buf, "%f", best)
}

// heap runs Heap's algorithm to generate permutations
func (b *BenchHaversine) heap(k int, arr []int, yield func()) {

	// Stop recursion and yield
	if k == 1 {
		yield()
		return
	}

	// Generate permutations with k-th unaltered
	// Initially k = length(A)
	b.heap(k-1, arr, yield)

	// Generate permutations for k-th swapped with each k-1 initial
	for i := 0; i < k-1; i++ {
		// Swap choice dependent on parity of k (even or odd)
		if k%2 == 0 {
			arr[i], arr[k-1] = arr[k-1], arr[i]
		} else {
			arr[0], arr[k-1] = arr[k-1], arr[0]
		}
		b.heap(k-1, arr, yield)
	}
}

// distance calculates the spherical distance between two cities
func (b *BenchHaversine) distance(i, j int) float64 {

	// No memoization here to benchmark floating point calculation

	c1 := &b.cities[i]
	lat1, lng1 := b.normalize(c1.lat, c1.lng)
	c2 := &b.cities[j]
	lat2, lng2 := b.normalize(c2.lat, c2.lng)

	d := b.haversine(lat1, lng1, lat2, lng2)
	return d
}

// normalize sanitizes the coordinates
func (b *BenchHaversine) normalize(lat, lng float64) (float64, float64) {

	lat = math.Mod(lat+90.0, 360.0) - 90.0
	if lat > 90.0 {
		lat = 180.0 - lat
		lng += 180.0
	}

	lng = math.Mod(lng+180.0, 360.0) - 180.0
	return lat, lng
}

// haversine calculates the spherical distance between two points
func (b *BenchHaversine) haversine(lat1, lng1, lat2, lng2 float64) float64 {

	const earth = 6371.0088
	square := func(x float64) float64 { return x * x }
	rad := func(x float64) float64 { return x * math.Pi / 180 }

	lat1, lng1 = rad(lat1), rad(lng1)
	lat2, lng2 = rad(lat2), rad(lng2)
	lat := lat2 - lat1
	lng := lng2 - lng1

	d := square(math.Sin(lat*0.5)) + math.Cos(lat1)*math.Cos(lat2)*square(math.Sin(lng*0.5))
	return earth * 2 * math.Asin(math.Sqrt(d))
}
