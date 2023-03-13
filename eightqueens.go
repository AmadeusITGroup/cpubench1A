package main

import (
	"fmt"
	"log"
	"strings"
)

// chessboard represents a chess board as 64 bits
type chessboard struct {
	x uint64
}

// isset returns true if a square is marked
func (cb *chessboard) isset(x, y int) bool {
	return cb.x&(1<<uint(y*8+x)) != 0
}

// set marks a square
func (cb *chessboard) set(x, y int) {
	cb.x |= 1 << uint(y*8+x)
}

// setxylines marks the board for a new queen (horizontal, vertical, diagonals)
func (cb *chessboard) setxylines(x, y int) {
	for i := 0; i < 8; i++ {
		xl, yl := uint64(1<<uint(x+i*8)), uint64(1<<uint(i+y*8))
		xp, yp := x+i, y+i
		xm, ym := x-i, y-i
		var d1, d2, d3, d4 uint64
		if xp < 8 {
			if yp < 8 {
				d1 = uint64(1 << uint(xp+8*yp))
			}
			if ym >= 0 {
				d2 = uint64(1 << uint(xp+8*ym))
			}
		}
		if xm >= 0 {
			if yp < 8 {
				d3 = uint64(1 << uint(xm+8*yp))
			}
			if ym >= 0 {
				d4 = uint64(1 << uint(xm+8*ym))
			}
		}
		cb.x |= (xl | yl) | (d1 | d2) | (d3 | d4)
	}
}

// display is used to display a chessboard for debugging purposes
func (cb *chessboard) display() {
	var sb strings.Builder
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			if cb.isset(i, j) {
				sb.WriteByte('X')
			} else {
				sb.WriteByte('.')
			}
		}
		sb.WriteByte('\n')
	}
	fmt.Print(sb.String())
}

// Bench8Queens is a 8 queens problem solver using a brute force approach.
// No memory allocation, but a recursive algorithm with a lot of bit twiddling.
type Bench8Queens struct {
	res []chessboard
}

// NewBench8Queens allocates a new benchmark object
func NewBench8Queens() *Bench8Queens {
	return &Bench8Queens{
		res: make([]chessboard, 0, 100),
	}
}

// Run solves the 8 queens problem 4 times (arbitrary)
func (b *Bench8Queens) Run() {
	for i := 0; i < 4; i++ {
		var c chessboard
		b.rowIterate(c, c, 0)
		if len(b.res) != 92 {
			log.Fatal("8 queens problem solutions mismatch")
		}
		b.res = b.res[:0]
	}
}

// rowIterate recursively checks the positions for a given row.
// The stack is used to backtrack in case of faulty position.
// We store the positions of the queens, plus a mask marking faulty positions.
func (b *Bench8Queens) rowIterate(c chessboard, mask chessboard, x int) {
	if x == 8 {
		b.res = append(b.res, c)
		return
	}
	for y := 0; y < 8; y++ {
		if mask.isset(x, y) {
			continue
		}
		ccur, mcur := c, mask
		ccur.set(x, y)
		mcur.setxylines(x, y)
		b.rowIterate(ccur, mcur, x+1)
	}
}
