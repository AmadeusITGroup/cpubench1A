package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

// Version of the program
const Version = "3.1"

// Definition of the command line flags
var (
	flagWorkers  = flag.Int("workers", -1, "Number of workers. Default is 4*threads")
	flagThreads  = flag.Int("threads", -1, "Number of Go threads (i.e. GOMAXPROCS). Default is all OS processors")
	flagRun      = flag.Bool("run", false, "Run a single benchmark iteration")
	flagRunOLTP  = flag.Bool("runoltp", false, "Run a single iteration of the OLTP benchmark")
	flagBench    = flag.Bool("bench", false, "Run standard benchmark (multiple iterations)")
	flagFreq     = flag.Bool("freq", false, "Measure the frequency of the CPU")
	flagOLTP     = flag.Bool("oltp", false, "Run OLTP benchmark (multiple iterations)")
	flagTPS      = flag.Int("tps", 100, "Target throughput of OLTP benchamrk")
	flagDuration = flag.Int("duration", 60, "Duration in seconds of a single iteration")
	flagNb       = flag.Int("nb", 10, "Number of iterations")
	flagRes      = flag.String("res", "", "Optional result append file")
	flagVersion  = flag.Bool("version", false, "Display program version and exit")
)

// main entry point of the progam
func main() {

	flag.Parse()

	// Fix number of of threads of the Go runtime.
	// By default, 4 workers per thread.
	if *flagThreads == -1 {
		*flagThreads = runtime.NumCPU()
	}
	if *flagWorkers == -1 {
		*flagWorkers = *flagThreads * 4
	}
	runtime.GOMAXPROCS(*flagThreads)

	// Run a single iteration or a full benchmark
	var err error
	switch {
	case *flagRun:
		err = runBench(injectSaturation)
	case *flagRunOLTP:
		err = runBench(injectOLTP)
	case *flagBench:
		err = stdBench()
	case *flagOLTP:
		err = oltpBench()
	case *flagFreq:
		err = measureFreq()
	case *flagVersion:
		err = displayVersion()
	default:
		flag.Usage()
		os.Exit(-1)
	}

	if err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}

// Injector is an injection policy
type Injector func(chan WorkerOp, chan bool)

// runBench runs a simple benchmark
func runBench(inject Injector) error {

	log.Printf("CPU benchmark with %d threads and %d workers", *flagThreads, *flagWorkers)

	// We will maintain the workers busy by pre-filling a buffered channel
	init := make(chan WorkerOp, *flagWorkers)
	input := make(chan WorkerOp, *flagWorkers*32)
	output := make(chan int, *flagWorkers)

	log.Printf("Initializing workers")

	// Spawn workers and trigger initialization
	workers := []*Worker{}
	for i := 0; i < *flagWorkers; i++ {
		w := NewWorker(i, init, input, output)
		workers = append(workers, w)
		go w.Run()
		init <- OpInit
	}

	// Wait for all workers to be initialized
	for range workers {
		<-output
	}

	// Run a synchronous garbage collection now to avoid processing the garbage
	// associated to the initialization during the benchmark
	runtime.GC()
	runtime.GC()

	// Start the benchmark: it will run for a given duration
	log.Printf("Start")
	begin := time.Now()
	stop := make(chan bool)
	time.AfterFunc(time.Duration(*flagDuration)*time.Second, func() {
		log.Printf("Stop signal")
		stop <- true
	})

	// Apply the injection
	inject(input, stop)

	// Signal the end of the benchmark to workers, and aggregate results
	for range workers {
		input <- OpExit
	}
	nb := 0
	for range workers {
		nb += <-output
	}
	end := time.Now()
	log.Printf("End")

	// Calculate resulting throughput
	ns := float64(end.Sub(begin).Nanoseconds())
	res := float64(nb) * 1000000000.0 / ns
	log.Printf("THROUGHPUT %.6f", res)
	if *flagRes != "" {
		if err := AppendResult(*flagRes, *flagWorkers, res); err != nil {
			log.Print(err)
			log.Printf("Cannot write result into temporary file: %s", *flagRes)
		}
	}
	log.Print()

	return nil
}

// injectSaturation injects traffic by saturating the input queue.
// It is used for the standard benchmark.
func injectSaturation(input chan WorkerOp, stop chan bool) {

	// Saturation benchmark loop: we avoid checking for the timeout too often
	for {
		select {
		case <-stop:
			return
		default:
			for i := 0; i < *flagWorkers*16; i++ {
				input <- OpStep
			}
		}
	}
}

// injectOLTP injects traffic by limiting the input throughput.
// It is used for the OLTP benchmark.
func injectOLTP(input chan WorkerOp, stop chan bool) {

	// Calculate a suitable period and number of transactions per period
	var period int
	switch {
	case *flagTPS < 10:
		period = 1000
	case *flagTPS < 100:
		period = 100
	case *flagTPS < 500:
		period = 50
	case *flagTPS < 1000:
		period = 20
	default:
		period = 10
	}
	nbPeriods := 1000 / period
	nbTrans := *flagTPS / nbPeriods

	// Rounding errors need to be corrected, so the first period is adjusted with a bit more transactions
	nbTransFirst := *flagTPS - nbTrans*nbPeriods
	if nbTransFirst < 0 {
		nbTransFirst = 0
	}

	log.Printf("Injection: %d transactions every %d ms for %d periods/s", nbTrans, period, nbPeriods)
	log.Printf("Injection correction: %d", nbTransFirst)
	ticker := time.Tick(time.Duration(period) * time.Millisecond)
	iPeriod := 0

	// Inject nbTrans transactions for each period
	for {
		select {
		case <-stop:
			return
		case <-ticker:
			n := nbTrans
			if iPeriod == 0 {
				n += nbTransFirst
			}
			for i := 0; i < n; i++ {
				input <- OpStep
			}
			iPeriod++
			if iPeriod == nbPeriods {
				iPeriod = 0
			}
		}
	}
}

// stdBench runs multiple benchmarks (single-threaded and then multi-threaded)
func stdBench() error {

	// Display CPU information
	log.Println("Version: ", Version)
	log.Print()
	if err := displayCPU(); err != nil {
		return nil
	}

	// Create a temporary file storing the results
	tmp, err := os.CreateTemp("", "cpubench1a-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	// Run multiple benchmarks in sequence
	log.Print("Single threaded performance")
	log.Print("===========================")
	log.Print()
	for i := 0; i < *flagNb; i++ {
		if err := spawnBench(1, tmp.Name()); err != nil {
			return err
		}
	}

	// Run multiple benchmarks in sequence
	log.Print("Multi-threaded performance")
	log.Print("==========================")
	log.Print()
	for i := 0; i < *flagNb; i++ {
		if err := spawnBench(*flagWorkers, tmp.Name()); err != nil {
			return err
		}
	}

	// Display statistics from the temporary file
	DisplayResult(tmp, *flagWorkers)
	tmp.Close()
	return nil
}

// oltpBench runs multiple OLTP benchmark progressively increasing the throughput.
// Purpose is to measure the CPU usage for a give throughput.
func oltpBench() error {

	// Display CPU information
	log.Println("Version: ", Version)
	log.Print()
	if err := displayCPU(); err != nil {
		return nil
	}

	log.Print("OLTP benchmark")
	log.Print("==============")
	log.Print()

	// CPU usage is calculated from the last call to cpu.Percent.
	// This is the initial call.
	if _, err := cpu.Percent(0, false); err != nil {
		return err
	}

	// We run a few more iterations to try to saturate the CPU
	for i := 0; i < *flagNb+4; i++ {

		if i == 0 {
			// Used to measure CPU usage when nothing runs (zero throughput)
			time.Sleep(time.Duration(*flagDuration) * time.Second)
		} else {
			// No temp file to fetch results for now
			if err := spawnOLTP(i); err != nil {
				return err
			}
		}

		// This represents the average CPU usage percentage for the last iteration
		p, err := cpu.Percent(0, false)
		if err != nil {
			return err
		}
		log.Printf("CPU USAGE: %.3f", p[0])
		log.Print()
	}

	return nil
}

// spawnBench runs a benchmark as an external process
func spawnBench(workers int, resfile string) error {

	// Get executable path
	executable, err := os.Executable()
	if err != nil {
		return nil
	}

	// Build parameters
	opt := []string{
		"-run",
		"-threads", strconv.Itoa(*flagThreads),
		"-workers", strconv.Itoa(workers),
		"-duration", strconv.Itoa(*flagDuration),
		"-res", resfile,
	}

	// Execute command in blocking mode
	cmd := exec.Command(executable, opt...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// spawnOltp runs an OLTP benchmark as an external process
func spawnOLTP(it int) error {

	// Get executable path
	executable, err := os.Executable()
	if err != nil {
		return nil
	}

	// Build parameters
	opt := []string{
		"-runoltp",
		"-tps", strconv.Itoa(it * (*flagTPS) / (*flagNb)),
		"-threads", strconv.Itoa(*flagThreads),
		"-workers", strconv.Itoa(*flagWorkers),
		"-duration", strconv.Itoa(*flagDuration),
	}

	// Execute command in blocking mode
	cmd := exec.Command(executable, opt...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// displayCPU displays some CPU information
func displayCPU() error {

	ctx := context.Background()

	// Get type of CPU, frequency
	cpuinfo, err := cpu.InfoWithContext(ctx)
	if err != nil {
		return err
	}

	log.Printf("CPU: %s / %s", cpuinfo[0].VendorID, cpuinfo[0].ModelName)
	log.Printf("Max freq: %.2f mhz (as reported by OS)", cpuinfo[0].Mhz)

	// The core/thread count is wrong on some architectures
	nc, err := cpu.CountsWithContext(ctx, false)
	if err != nil {
		return err
	}
	nt, err := cpu.CountsWithContext(ctx, true)
	if err != nil {
		return err
	}

	log.Printf("Cores: %d", nc)
	log.Printf("Threads: %d", nt)

	// NUMA topology retrieval only works on Linux
	if files, err := filepath.Glob("/sys/devices/system/node/node[0-9]*/cpu[0-9]*"); err == nil && len(files) > 0 {

		// Fetch NUMA topology
		numa := map[int]int{}
		for _, f := range files {
			t := strings.Split(strings.TrimPrefix(f, "/sys/devices/system/node/"), "/")
			if len(t) > 1 {
				var n, c int
				if n, err = strconv.Atoi(strings.TrimPrefix(t[0], "node")); err != nil {
					continue
				}
				if c, err = strconv.Atoi(strings.TrimPrefix(t[1], "cpu")); err != nil {
					continue
				}
				numa[c] = n
			}
		}

		// Display NUMA topology
		for _, c := range cpuinfo {
			n, ok := numa[int(c.CPU)]
			if !ok {
				n = -1
			}
			log.Printf("CPU:%3d Socket:%3s CoreId:%3s Node:%3d", c.CPU, c.PhysicalID, c.CoreID, n)
		}
	}

	log.Print()
	return nil
}

// measureFreq attempts to measure the CPU frequency by counting CPU cycles
func measureFreq() error {

	// First run to warm the CPU
	log.Println("Version:", Version)
	log.Println("Warming-up CPU")
	CountASM(NFREQ)
	log.Println("Measuring ...")

	// Second run to perform the actual measurement
	t := time.Now()
	CountASM(NFREQ)
	t2 := time.Now()
	dur := t2.Sub(t).Seconds()

	// The loop contains 1024 dependent instructions (1 cycle per instruction)
	// plus a test/jump (resulting in 1 or 2 additional cycles)
	log.Println("Frequency:", float64(NFREQ)/1024.0*ASMLoopCycles/dur/1.0e9, "GHz")
	return nil
}

// displayVersion just prints the program version
func displayVersion() error {

	fmt.Println("Version:", Version)
	return nil
}
