package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
)

// ResultMap stores the results indexed by number of workers
type ResultMap map[int][]float64

// AppendResult writes the result at the end of the temporary file
func AppendResult(resfile string, workers int, result float64) error {

	f, err := os.OpenFile(resfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "%d %.6f\n", workers, result)
	return nil
}

// DisplayResult displays some statistics about the results
func DisplayResult(f *os.File, workers int) error {

	// Read temporary file
	m, err := readResult(f)
	if err != nil {
		return err
	}

	log.Print("Results")
	log.Print("=======")
	log.Print()

	// Display statistics on results
	displayStat("Single thread", m[1])
	displayStat("Multi-thread", m[workers])

	return nil
}

// readResult reads the temporary file and build a map of the results
func readResult(f *os.File) (ResultMap, error) {

	var workers int
	var throughput float64
	m := ResultMap{}

	// Scan the temporary file, one record by line
	f.Seek(0, 0)
	scan := bufio.NewScanner(f)
	for scan.Scan() {

		// Decode a single record
		n, err := fmt.Sscan(scan.Text(), &workers, &throughput)
		if err != nil || n != 2 {
			return nil, err
		}
		m[workers] = append(m[workers], throughput)
	}
	if err := scan.Err(); err != nil {
		return nil, err
	}

	return m, nil
}

// displayStat calculates basic statistics and displays them
func displayStat(title string, r []float64) {

	// Calculate min, max, average
	min, max, sum := r[0], r[0], r[0]
	for _, x := range r[1:] {
		if x < min {
			min = x
		}
		if x > max {
			max = x
		}
		sum += x
	}
	avg := sum / float64(len(r))

	// Calculate median
	sort.Float64s(r)
	var median float64
	if len(r)%2 == 0 {
		a, b := r[len(r)/2-1], r[len(r)/2]
		median = (a + b) / 2.0
	} else {
		median = r[len(r)/2]
	}

	// Display
	log.Print(title)
	log.Printf("    Minimum: %.6f", min)
	log.Printf("    Average: %.6f", avg)
	log.Printf("     Median: %.6f", median)
	log.Printf("    Maximum: %.6f", max)
	log.Print()
}
