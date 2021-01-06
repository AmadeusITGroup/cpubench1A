# cpubench1a - a CPU benchmark program

## Purpose

cpubench1a is a CPU benchmark whose purpose is to measure the global computing power of a Linux machine. It is used at [Amadeus](https://www.amadeus.com) to qualify bare-metal and virtual boxes, and compare generations of machines or VMs. It is delivered as static self-contained Go binaries for x86_64 and Aarch64 CPU architectures.

It runs a number of throughput oriented tests to establish:

 - an estimation of the throughput a single OS processor can sustain (i.e. a single-threaded benchmark)
 - an estimation of the throughput all the OS processors can sustain (i.e. a multi-threaded benchmark)

An OS processor is defined as an entry in the /proc/cpuinfo file. Depending on the hardware, hypervisor, and operating system configuration, it can correspond to a full CPU core, a vCPU, or just a hardware thread.

## Build and launch

It is recommended to build the binaries on a reference machine (eventually different from the machines to be compared by the benchmark). The idea is to use the same binaries on all the machines part of the benchmark to make sure the same exact code is run everywhere. We suggest to use any Linux x86_64 box supporting the Go toolchain (no specific constraint here), and build the ARM binary using cross compilation.

To build from source (for x86_64, from a x86_64 box):

```
go build
```

To build from source (for Aarch64, from a x86_64 box):

```
GOOS=linux GOARCH=arm64  go build
```

The resulting binary is statically linked and not dependent on the Linux distribution or version (except a minimal 2.6.23 kernel version). In order to run the benchmark, the binary can be directly copied to the target machines, and run there. 

The command line parameters are:

```
Usage of ./cpubench1a:
  -bench
    	Run standard benchmark (multiple iterations). Mutually exclusive with -run
  -duration int
    	Duration in seconds of a single iteration (default 60)
  -nb int
    	Number of iterations (default 5)
  -run
    	Run a single benchmark iteration. Mutually exclusive with -bench
  -threads int
    	Number of Go threads (i.e. GOMAXPROCS). Default is all OS processors (default -1)
  -workers int
    	Number of workers. Default is 4*threads (default -1)
```

The canonical way to launch the benchmark is just:

```
$ ./cpubench1a -bench
```

By default, it runs for a bit more than 10 minutes (5 iterations of 60 seconds each for single and multiple threads). The default number of threads is the number of OS processors, and the default number of workers is 4 times the number of threads.

Before launching the tests, the program displays some information about the CPU extracted from /proc/cpuinfo and the NUMA topology of the system (if available).

## Principle

The principle is very similar to SPECint or Coremark integer benchmarks. It is basically a loop on multiple algorithms whose execution time has been balanced. The algorithms are chosen to be independent, quick to execute, but complex enough to represent real-world code. They include:

- deflate compression/decompression + base64 encoding/decoding
- sorting a set of records in multiple orders
- parsing and executing short awk programs
- JSON messages parsing and encoding
- building/using some btree data structures
- a Monte-Carlo simulation calculating the availability of a NoSQL cluster

These algorithms are not specifically representative of a given Amadeus application or functional transaction. Compression/decompression, encoding/decoding, data structures management, sorting small datasets are typical of back-end software though.

To check the benchmark is relevant (and the execution time of one algorithm does not dwarf all the other ones), the relative execution time of the various algorithms can be displayed using:

```
$ go test -bench=.
goos: linux
goarch: amd64
pkg: cpubench1a
BenchmarkCompression-12    	    2732	    413155 ns/op
BenchmarkAwk1-12           	    3524	    331972 ns/op
BenchmarkAwk2-12           	    3718	    425116 ns/op
BenchmarkJson-12           	    2221	    515674 ns/op
BenchmarkBtree-12          	    3807	    313914 ns/op
BenchmarkSort-12           	    2049	    539166 ns/op
BenchmarkSimulation-12     	    1812	    661699 ns/op
PASS
ok  	cpubench1a	8.931s
```

The architecture of the benchmark program is the following. There are a main driver and multiple workers. The driver is pushing transactions to a queue. Each worker fetches transactions from the queue, and executes them. Each transaction executes the above algorithms (all of them). The implementation of these algorithms has been designed to be independent from the context (i.e. reentrant, no contention on shared data), and CPU bound. The queuing/dequeuing overhead is negligible compared to the transaction execution time. The queue is saturated for all the benchmark duration except at the end, so there is no wait state in the workers.

A single-threaded run only involves a single worker. A multi-threaded run involves 4 workers per OS processors (by default).

Each test iteration runs for a given duration (typically 1 minute). The score of the benchmark is simply the number of transaction executions per second.

A normal benchmark run involves 5 test iterations in single-threaded mode, and 5 test iterations in multi-threaded mode. The resulting score is defined as the **maximum** reported throughput in each category. Considering the maximum aims to counter the negative effect of CPU throttling, variable frequencies and noisy neighbours on some environments.

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

## Versioning

Because the purpose of this software is to compare the CPU efficiency of various systems, the resulting scores are only meaningful for a given version of the software compiled with a given version of the Go compiler.

| Version | Go compiler   |
|---------|---------------|
| 1.0     | 1.5.2         |

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



