package main

import (
	"log"
)

// Benchmark is just a runnable thing
type Benchmark interface {
	Run()
}

// WorkerOp is an enumerate representing the type of operations processed by the workers
type WorkerOp byte

const (
	OpNull = iota
	OpInit
	OpStep
	OpExit
)

// Worker does represent a single worker
type Worker struct {
	id         int
	init       chan WorkerOp
	input      chan WorkerOp
	output     chan int
	nb         int
	benchmarks []Benchmark
}

// NewWorker creates a worker
func NewWorker(id int, init chan WorkerOp, input chan WorkerOp, output chan int) *Worker {
	return &Worker{
		id:     id,
		init:   init,
		input:  input,
		output: output,
	}
}

// Run is triggered when the worker is started
func (w *Worker) Run() {

	// Initialize the worker
	<-w.init
	w.Init()

	// Main worker loop, fetching operations from the input channel
	for op := range w.input {
		switch op {
		case OpStep:
			w.Step()
			w.nb++
		case OpExit:
			w.Exit()
			return
		default:
			log.Printf("Wrong operation %d", op)
		}
	}
}

// Init performs worker initialization
func (w *Worker) Init() {

	// Here are the algorithms that the worker runs for 1 transaction
	w.benchmarks = []Benchmark{
		NewBenchCompression(),
		NewBenchSort(),
		NewBenchAwk(),
		NewBenchJson(),
		NewBenchBtree(),
		NewBenchSimulation(),
		NewBench8Queens(),
		NewBenchMemory(),
		NewBenchImage(),
		NewBenchCrypto(),
		NewBenchPearls(),
		NewBenchGraph(),
		NewBenchLogging(),
	}
	w.output <- 0
}

// Step executes one transaction (i.e. one step)
func (w *Worker) Step() {
	for _, x := range w.benchmarks {
		x.Run()
	}
}

// Exit sends the throughput of the worker back to the driver
func (w *Worker) Exit() {
	w.output <- w.nb
}
