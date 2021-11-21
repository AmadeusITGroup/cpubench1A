# cpubench1a - a CPU benchmark program

## Purpose

cpubench1a is a CPU benchmark whose purpose is to measure the global computing power of a Linux machine. It is used at [Amadeus](https://www.amadeus.com) to qualify bare-metal and virtual boxes, and compare generations of machines or VMs. It is delivered as static self-contained Go binaries for x86_64 and Aarch64 CPU architectures.

It runs a number of throughput oriented tests to establish:

 - an estimation of the throughput a single OS processor can sustain (i.e. a single-threaded benchmark)
 - an estimation of the throughput all the OS processors can sustain (i.e. a multi-threaded benchmark)

An OS processor is defined as an entry in the /proc/cpuinfo file. Depending on the hardware, hypervisor, and operating system configuration, it can correspond to a full CPU core, a vCPU, or just a hardware thread.

## Build and launch

The easiest way to run this tool is to directly download the binaries from github (see the releases). If this is not possible or acceptable, it is easy enough to build it, provided the correct version of the Go compiler is installed.

It is recommended to build the binaries on a reference machine (eventually different from the machines to be compared by the benchmark). The idea is to use the same binaries on all the machines part of the benchmark to make sure the same exact code is run everywhere. We suggest to use any Linux x86_64 box supporting the Go toolchain (no specific constraint here), and build the ARM binary using cross compilation.

To build from source (for x86_64, from a x86_64 box):

```
CGO_ENABLED=0 go build
```

To build from source (for Aarch64, from a x86_64 box):

```
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build
```

The resulting binary is statically linked and not dependent on the Linux distribution or version (except a minimal 2.6.23 kernel version). In order to run the benchmark, the binary can be directly copied to the target machines, and run there. 

The command line parameters are:

```
Usage of ./cpubench1a:
  -bench
    	Run standard benchmark (multiple iterations). Mutually exclusive with -run
  -duration int
    	Duration in seconds of a single iteration (default 60)
  -freq
    	Measure the frequency of the CPU
  -nb int
    	Number of iterations (default 10)
  -res string
    	Optional result append file
  -run
    	Run a single benchmark iteration. Mutually exclusive with -bench
  -threads int
    	Number of Go threads (i.e. GOMAXPROCS). Default is all OS processors (default -1)
  -version
    	Display program version and exit
  -workers int
    	Number of workers. Default is 4*threads (default -1)
```

The canonical way to launch the benchmark is just:

```
$ ./cpubench1a -bench
```

By default, it runs for a bit more than 20 minutes (10 iterations of 60 seconds each for single and multiple threads). The default number of threads is the number of OS processors, and the default number of workers is 4 times the number of threads.

Before launching the tests, the program displays some information about the CPU extracted from /proc/cpuinfo and the NUMA topology of the system (if available).

## Principle

The principle is very similar to SPECint or Coremark integer benchmarks. It is basically a loop on multiple algorithms whose execution time has been balanced. The algorithms are chosen to be independent, quick to execute, but complex enough to represent real-world code. They include:

- deflate compression/decompression + base64 encoding/decoding
- sorting a set of records in multiple orders
- parsing and executing short awk programs
- JSON messages parsing and encoding
- building/using some btree data structures
- a Monte-Carlo simulation calculating the availability of a NoSQL cluster
- a 8 queens chess problem solver
- sequential buffer building with scattered memory access patterns
- small image composition and jpeg encoding
- basic cryptography exercising 3DES/CTR algorithms
- solving Dijkstra's pearls problem

These algorithms are not specifically representative of a given Amadeus application or functional transaction. Compression/decompression, encoding/decoding, crypto, data structures management, sorting small datasets, buffer building from scattered memory accesses are typical of back-end software though. We do not really care about the absolute throughput of each individual algorithm, but rather about the transactional throughput, each transaction being a sequence involving all the algorithms, each of them run on a small working set.

To check the benchmark is relevant (and the execution time of one algorithm does not dwarf all the other ones), the relative execution time of the various algorithms can be displayed using:

```
$ go test -bench=. -benchmem
goos: linux
goarch: amd64
pkg: cpubench1a
cpu: Intel(R) Core(TM) i7-8850H CPU @ 2.60GHz
BenchmarkCompression-12    	    3049	    386162 ns/op	   45490 B/op	      17 allocs/op
BenchmarkAwk1-12           	    4596	    271159 ns/op	   37482 B/op	     405 allocs/op
BenchmarkAwk2-12           	    7002	    212586 ns/op	  114376 B/op	     910 allocs/op
BenchmarkJson-12           	    1988	    524209 ns/op	    8680 B/op	      89 allocs/op
BenchmarkBtree1-12         	    8228	    143678 ns/op	    9783 B/op	       4 allocs/op
BenchmarkBtree2-12         	    5002	    238617 ns/op	   13117 B/op	       4 allocs/op
BenchmarkSort-12           	    2649	    434825 ns/op	     158 B/op	       4 allocs/op
BenchmarkSimulation-12     	    1982	    602981 ns/op	   28985 B/op	    1218 allocs/op
Benchmark8Queens-12        	    2804	    408413 ns/op	       0 B/op	       0 allocs/op
BenchmarkMemory-12         	    3924	    260379 ns/op	    6131 B/op	       0 allocs/op
BenchmarkImage-12          	    2012	    559858 ns/op	     524 B/op	       9 allocs/op
BenchmarkCrypto-12         	    3314	    366158 ns/op	    1376 B/op	      11 allocs/op
BenchmarkPearls-12         	    3771	    303114 ns/op	       0 B/op	       0 allocs/op
BenchmarkAll-12            	     247	   4652340 ns/op	  366185 B/op	    2700 allocs/op
PASS
ok  	cpubench1a	17.501s
```

Each individual algorithm should represent only a fraction of the CPU consumption of the total (BenchmarkAll).

We try to make sure that each workload does not allocate too much in order to avoid benchmarking the garbage collector instead of the actual algorithms.

The architecture of the benchmark program is the following. There are a main driver and multiple workers. The driver is pushing transactions to a queue. Each worker fetches transactions from the queue, and executes them. Each transaction executes the above algorithms (all of them). The implementation of these algorithms has been designed to be independent from the context (i.e. reentrant, no contention on shared data), and CPU bound. The queuing/dequeuing overhead is negligible compared to the transaction execution time. The queue is saturated for all the benchmark duration except at the end, so there is no wait state in the workers.

A single-threaded run only involves a single worker. A multi-threaded run involves 4 workers per OS processors (by default).

Each test iteration runs for a given duration (typically 1 minute). The score of the benchmark is simply the number of transaction executions per second.

A normal benchmark run involves 10 test iterations in single-threaded mode, and 10 test iterations in multi-threaded mode. Statistics about the single-threaded and multi-threaded runs are given at the end.The resulting score is defined as the **maximum** reported throughput in each category. Considering the maximum aims to counter the negative effect of CPU throttling, variable frequencies and noisy neighbours on some environments.

```
2021/05/04 21:32:08 Results
2021/05/04 21:32:08 =======
2021/05/04 21:32:08 
2021/05/04 21:32:08 Single thread
2021/05/04 21:32:08     Minimum: 203.225590
2021/05/04 21:32:08     Average: 203.990533
2021/05/04 21:32:08      Median: 204.007989
2021/05/04 21:32:08     Maximum: 204.833596
2021/05/04 21:32:08 
2021/05/04 21:32:08 Multi-thread
2021/05/04 21:32:08     Minimum: 976.563635
2021/05/04 21:32:08     Average: 1060.236858
2021/05/04 21:32:08      Median: 1039.998520
2021/05/04 21:32:08     Maximum: 1156.083724
2021/05/04 21:32:08 
```

## Rationale: avoiding pitfalls

We have decided to write our own benchmark to avoid the following issues:

- we wanted to test CPUs only, and not operating systems, compilers, system libraries, runtime, etc ... of the tested platform. Of course, we still depend on the Go compiler/runtime, but it is supposed to be the same compiler/runtime for all the tests.

- we wanted to compare various physical or virtual on-premises machines, but also public cloud boxes. Some machines use a fixed CPU frequency. Some others use a variable CPU frequency and the guest operating system is aware. Some others use a variable CPU frequency at hypervisor level, but the guest operating system is not aware. Most CPUs adapt the frequency to their workload. By design, this benchmark presents meaningful figures whatever the situation.

- we wanted to mitigate the risk of having a vendor optimizing its offer to shine in a well-known industry benchmark.

The benchmark is therefore coded in pure Go, compiled with a specific Go version on a given reference machine, and linked statically. The binaries are provided for Intel/AMD and ARM (64 bits). They are copied to the machine we want to test, so we are sure the same code is executed whatever the Linux distribution/version/flavor. At the moment, the benchmark does only run under Linux, and no other operating system.

Limiting all memory allocations while testing real-world code is difficult. Each algorithm generates some memory allocations, and therefore some garbage collection activity. We have just ensured that the garbage collection cost is low compared to the runtime cost of the algorithms. Furthermore, each test iteration runs in a separate process, so that the memory accumulated by a given iteration does not degrade the garbage collection of the next iteration.

The variability of the results especially on non isolated environments (clouds) was a concern. We have not found any better mitigation mechanism than running multiple iterations of the same test and consider the maximum score. The benchmark is known to be sensitive to:

 - noisy neighbors running on the same hypervisor
 - CPU throttling, P-state/C-state configuration, at guest OS or hypervisor level
 - the NUMA configuration, and how the threads are distributed over the NUMA nodes

The benchmark measures throughput on single-threaded and multi-threaded code, so that we have a score bound by the maximum frequency a CPU core can get (for single-threaded tests), and the maximum frequency **all** the cores can get (for multi-threaded tests) - which can be different.

Each test is run multiple times (we suggest 5 times as a minimum), so that the system has time to set the maximum possible frequency, and to mitigate the variability of the performance and noisy neighbour effects. Each test runs in a separate process and starts from the same memory state to avoid impacts due to the non deterministic nature of memory garbage collection. The more runs, the better accuracy of the result.

## How to use the results?

The multi-threaded score is a good indicator of the relative power of CPU models for capacity/planning purposes. It can be used to support large-scale hardware footprint estimations.

Here is an example.

Let's suppose on VM A with 48 vCPU, we have a multi-threaded score of 5400. VM A can deliver 5400 / 48 = 112.5 / vCPU.

Let's suppose on VM B with 64 vCPU, we have a multi-threaded score of 6100. VM B can deliver 6100 / 64 = 95.31 / vCPU.

The relative power of vCPU between VM A and B is 112.5 / 95.31 = 1.18 in favor of VM A.

We have a CPU bound workload running on 2000 VM of type A. How many VM of type B do we need to cover the same workload? We need 2000 * 48 * 1.18 / 64 = 1770 VMs.

## CPU frequency measurement

This tool also supports a CPU frequency measuremnet mechanism. It can be launched using: 

```
$ ./cpubench1a -freq
```

It should execute whithin a few seconds.

The frequency is measured by counting the number of CPU cycles of a pre-defined loop, so it is independent from any system information exposed by the OS or the hypervisor. It has the benefit to measure a meaningful value, even if the hypervisor is lying to the guest OS.

The command can be launched once (with no other activity on the machine) to measure the maximum frequency for one core. It can be launched multiple times in parallel to measure the maximum frequency when multiple cores are active (which can be different, due to CPU power management features).

## Versioning

Because the purpose of this software is to compare the CPU efficiency of various systems, the resulting scores are only meaningful for a given version of the software compiled with a given version of the Go compiler.

| Version | Go compiler   |
|---------|---------------|
| 1.0     | 1.15.2        |
| 2.0     | 1.16.3        |
| 3.0     | 1.17.3        |

The scores measured with different versions of this benchmark MUST NOT be compared.

## Credits

Many thanks to the authors of the following packages:

|Package                         |Author          |License     |
|--------------------------------|----------------|------------|
|github.com/benhoyt/goawk        |Ben Hoyt        |MIT         |
|github.com/shirou/gopsutil/v3   |Shirou Wakayama |New BSD     |
|github.com/tidwall/btree        |Josh Baker      |MIT         |
|github.com/tidwall/gjson        |Josh Baker      |MIT         |

## License

This software is under MIT License.
