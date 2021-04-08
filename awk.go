package main

import (
	"bytes"
	"log"

	"github.com/benhoyt/goawk/interp"
	"github.com/benhoyt/goawk/parser"
)

// BenchAwk is a benchmark around Awk code interpretation
type BenchAwk struct {
	res bytes.Buffer
}

// awkPROG1 is a piece of AWK Mandelbrot calculation code.
// Copied and modified from https://github.com/benhoyt/goawk/blob/master/examples/mandel.awk
var awkPROG1 = []byte(`
BEGIN {
    width = 20; height = 15
    min_x = -2.1; max_x = 0.6
    min_y = -1.2; max_y = 1.2
    iters = 2

    colors[0] = "."
    colors[1] = "-"
    colors[2] = "+"
    colors[3] = "*"
    colors[4] = "%%"
    colors[5] = "#"
    colors[6] = "$"
    colors[7] = "@"
    colors[8] = " "

    inc_y = (max_y-min_y) / height
    inc_x = (max_x-min_x) / width
    y = min_y
    for (row=0; row<height; row++) {
        x = min_x
        for (col=0; col<width; col++) {
            zr = zi = 0
            for (i=0; i<iters; i++) {
                old_zr = zr
                zr = zr*zr - zi*zi + x
                zi = 2*old_zr*zi + y
                if (zr*zr + zi*zi > 4) break
            }
            printf colors[int(i*8/iters)]
            x += inc_x
        }
        y += inc_y
        printf "\n"
    }
}
`)

// awkPROG2 is a filtering AWK program
var awkPROG2 = []byte(`
/"Ai.*rt"/ { a++ }
/"C..celled"/ { c += $2 }
END { print "Resulat", a, c }
`)

// NewBenchAwk allocates a new benchmark object
func NewBenchAwk() *BenchAwk {
	return &BenchAwk{}
}

// Run parses and executes the two AWK programs
func (b *BenchAwk) Run() {
	b.test1()
	b.test2()
}

// test1 generates the Mandelbrot set by parsing and executing some AWK code
func (b *BenchAwk) test1() {
	prog, err := parser.ParseProgram(awkPROG1, nil)
	if err != nil {
		log.Fatal(err)
	}
	config := &interp.Config{
		Stdin:  bytes.NewReader(jsonAirlinesB),
		Output: &b.res,
		Vars:   []string{"OFS", ":"},
	}
	_, err = interp.ExecProgram(prog, config)
	if err != nil {
		log.Fatal(err)
	}
	b.res.Reset()
}

// test2 filters some input data by running a AWK program
func (b *BenchAwk) test2() {
	prog, err := parser.ParseProgram(awkPROG2, nil)
	if err != nil {
		log.Fatal(err)
	}
	config := &interp.Config{
		Stdin:  bytes.NewReader(jsonAirlinesB[:20000]),
		Output: &b.res,
		Vars:   []string{"FS", ":"},
	}
	_, err = interp.ExecProgram(prog, config)
	if err != nil {
		log.Fatal(err)
	}
	b.res.Reset()
}
