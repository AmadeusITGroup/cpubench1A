package main

import (
	"testing"
)

func BenchmarkCompression(b *testing.B) {
	x := NewBenchCompression()
	for n := 0; n < b.N; n++ {
		x.Run()
	}
}

func BenchmarkAwk1(b *testing.B) {
	x := NewBenchAwk()
	for n := 0; n < b.N; n++ {
		x.test1()
	}
}

func BenchmarkAwk2(b *testing.B) {
	x := NewBenchAwk()
	for n := 0; n < b.N; n++ {
		x.test2()
	}
}

func BenchmarkJson(b *testing.B) {
	x := NewBenchJson()
	for n := 0; n < b.N; n++ {
		x.Run()
	}
}

func BenchmarkBtree1(b *testing.B) {
	x := NewBenchBtree()
	for n := 0; n < b.N; n++ {
		x.test1()
	}
}

func BenchmarkBtree2(b *testing.B) {
	x := NewBenchBtree()
	for n := 0; n < b.N; n++ {
		x.test2()
	}
}

func BenchmarkSort(b *testing.B) {
	x := NewBenchSort()
	for n := 0; n < b.N; n++ {
		x.Run()
	}
}

func BenchmarkSimulation(b *testing.B) {
	x := NewBenchSimulation()
	for n := 0; n < b.N; n++ {
		x.Run()
	}
}

func Benchmark8Queens(b *testing.B) {
	x := NewBench8Queens()
	for n := 0; n < b.N; n++ {
		x.Run()
	}
}

func BenchmarkMemory(b *testing.B) {
	x := NewBenchMemory()
	for n := 0; n < b.N; n++ {
		x.Run()
	}
}

func BenchmarkImage(b *testing.B) {
	x := NewBenchImage()
	for n := 0; n < b.N; n++ {
		x.Run()
	}
}
