package main

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"sync"
	"time"
)

// BenchLogging is a log formatting benchmark with concealment and deduplication
type BenchLogging struct {
	mutex  sync.Mutex
	pool   [][]byte
	queue  chan []byte
	ack    chan bool
	t      time.Time
	output bytes.Buffer
	stats  map[string]int
	re     *regexp.Regexp
	last   LogEntry
	cnt    int
}

// LoggingEntry corresponds to one line of the log
type LogEntry struct {
	file string
	line int
	sev  string
	msg  string
}

// NewBenchLogging allocates a new benchmark object
func NewBenchLogging() *BenchLogging {
	return &BenchLogging{
		pool:  make([][]byte, 0, 32),
		ack:   make(chan bool),
		stats: make(map[string]int, 10),
		re:    regexp.MustCompile(`[\d]{4}-[\d]{4}-[\d]{4}-[\d]{2}`),
	}
}

// Run does log a few lines and format them
func (b *BenchLogging) Run() {

	// Start from an arbitrary date
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		log.Fatal(err)
	}
	b.t = time.Date(1972, 1, 16, 10, 20, 0, 0, loc)
	b.queue = make(chan []byte, 8)

	// This goroutine simulates log processing (buffering, concealment and statistics)
	go b.processLogs()

	// Log a bunch of lines mixing duplicates and lines with credit cards
	for i := 0; i < 16; i++ {
		b.log("checkin.cpp", 10+i, "INFO", "first\tfirst")
	}
	b.log("secu.cpp", 99, "INFO", "FYI, my credit card number is 1234-4321-1234-21 - feel free to use it")
	for i := 0; i < 32; i++ {
		b.log("boarding.cpp", 100, "WARN", "second\nsecond")
	}
	b.log("secu.cpp", 999, "INFO", "The credit card of my neighbor is 8792-4567-4321-22. Am I PCI-DSS compliant?")
	for i := 0; i < 64; i++ {
		b.log("checkout.cpp", 1000+i, "TRACE", "third\"third")
	}
	b.log("secu.cpp", 9999, "INFO", "Do not forget to help yourself using the 9001-1493-4378-23 credit card")
	for i := 0; i < 128; i++ {
		b.log("reporting.cpp", 10000, "INFO", "fourth\r\nfourth")
	}
	b.log("secu.cpp", 99999, "INFO", "Alarm! Hide this 1234-1234-1234-12 credit card")

	// Signal the end and wait for the goroutine termination
	b.flush()
	close(b.queue)
	<-b.ack
}

// get returns a buffer to be used for log formatting
func (b *BenchLogging) get() []byte {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if len(b.pool) == 0 {
		return make([]byte, 0, 256)
	}
	n := len(b.pool)
	ret := b.pool[n-1]
	b.pool = b.pool[:n-1]
	return ret
}

// put has to be called when the logging buffer can be discarded
func (b *BenchLogging) put(buf []byte) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	buf = buf[:0]
	b.pool = append(b.pool, buf)
}

// log adds a lot entry applying deduplication for juxtaposed entries
func (b *BenchLogging) log(file string, line int, sev string, msg string) {
	e := LogEntry{file, line, sev, msg}
	if e == b.last {
		b.cnt++
	} else {
		b.flush()
		b.logEntry(e)
		b.last = e
	}
}

// flush should be called to force the log entry to be processed
func (b *BenchLogging) flush() {
	if b.cnt > 0 {
		b.cnt++
		repeat := b.last
		repeat.msg = fmt.Sprintf("... repeated %d times ...", b.cnt)
		b.logEntry(repeat)
		b.cnt = 0
	}
}

// logEntry is a lower level method to add a log entry
func (b *BenchLogging) logEntry(e LogEntry) {
	b.t = b.t.Add(1 * time.Second)
	buf := b.get()
	buf = b.t.AppendFormat(buf, time.RFC3339Nano)
	buf = append(buf, ' ')
	buf = append(buf, e.sev...)
	buf = append(buf, ' ')
	buf = append(buf, e.file...)
	buf = append(buf, ':')
	buf = strconv.AppendInt(buf, int64(e.line), 10)
	buf = append(buf, ' ')
	buf = strconv.AppendQuote(buf, e.msg)
	b.queue <- buf
}

// processLogs is the main loop of the log processing goroutine
func (b *BenchLogging) processLogs() {

	b.output.Reset()

	for x := range b.queue {
		x = b.conceal(x)
		b.output.Write(x)
		b.output.WriteByte('\n')
		b.maintainStats(x)
		b.put(x)
	}
	b.ack <- true
}

// conceal detects credit card number and conceal them
func (b *BenchLogging) conceal(x []byte) []byte {
	if b.re.Find(x) == nil {
		return x
	}
	return b.re.ReplaceAll(x, []byte("XXXX-XXXX-XXXX-XX"))
}

// maintainStats counts the number of entries per severity
func (b *BenchLogging) maintainStats(x []byte) {
	n1 := bytes.IndexByte(x, ' ')
	if n1 == -1 {
		return
	}
	sev := x[n1+1:]
	n2 := bytes.IndexByte(sev, ' ')
	if n2 == -1 {
		return
	}
	sev = sev[:n2]
	b.stats[string(sev)] += 1
}
