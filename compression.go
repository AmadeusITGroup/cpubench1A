package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/hex"
	"hash/fnv"
	"io"
	"log"
	"unicode"
)

// BenchCompression is a hashing and zlib/base64 encoding/decoding benchmark
type BenchCompression struct {
	buf1 bytes.Buffer
	buf2 bytes.Buffer
	w    *zlib.Writer
}

// NewBenchCompression allocates a new benchmark object
func NewBenchCompression() *BenchCompression {
	return &BenchCompression{}
}

// Run the benchmark step
func (b *BenchCompression) Run() {

	// Calculate hash code of input
	h := fnv.New64a()
	h.Write(jsonPopulationB)
	h1 := h.Sum64()

	// Compress the input
	if b.w == nil {
		b.w = zlib.NewWriter(&b.buf1)
	} else {
		b.w.Reset(&b.buf1)
	}
	b.w.Write(jsonPopulationB)
	b.w.Close()

	// Encode the compressed result in base64
	encoder := base64.NewEncoder(base64.StdEncoding, &b.buf2)
	encoder.Write(b.buf1.Bytes())
	encoder.Close()

	// Decode the base64
	b.buf1.Reset()
	decoder := base64.NewDecoder(base64.StdEncoding, &b.buf2)
	io.Copy(&b.buf1, decoder)

	// Uncompress the decoded result
	b.buf2.Reset()
	r, err := zlib.NewReader(&b.buf1)
	if err != nil {
		log.Fatal("Error", err)
	}
	io.Copy(&b.buf2, r)
	r.Close()

	// Calculate a second hash code on the result
	h.Reset()
	h.Write(b.buf2.Bytes())
	h2 := h.Sum64()

	// The hash codes should be identical
	if h1 != h2 {
		log.Fatal("Hash mismatch error")
	}

	// Encode in hexa
	b.buf1.Reset()
	hexa := hex.NewEncoder(&b.buf1)
	io.Copy(hexa, &b.buf2)

	// Transform in UTF-8 uppercase
	slice := b.buf1.Bytes()
	for i, c := range slice {
		slice[i] = byte(unicode.ToUpper(rune(c)))
	}

	b.buf1.Reset()
	b.buf2.Reset()
}
