package main

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
	"io"
	"log"
)

const CRYPTO_KEY = "01234567890123456789ABCD"

// BenchCrypto is a symmetric cryptography benchmark.
// It explicitly uses outdated crypto algorithms making sure they are not implemented in assembly.
// We do not want to benchmark AES-NI or other hardware crypto support extensions,
// because their performance is too tied to the CPU architecture.
// We just use the 3DES block cipher with CTR mode of operation.
type BenchCrypto struct {
	input  []byte
	cipher cipher.Block
	out1   bytes.Buffer
	out2   bytes.Buffer
	buf    []byte
}

// NewBenchCrypto allocates a new benchmark object
func NewBenchCrypto() *BenchCrypto {

	// Triple DES: as insecure, triple the bloat
	c, err := des.NewTripleDESCipher([]byte(CRYPTO_KEY))
	if err != nil {
		panic(err)
	}

	// Build a dummy plaintext
	in := []byte{}
	for i := 0; i < 6000; i++ {
		in = append(in, byte(i))
	}

	return &BenchCrypto{
		input:  in,
		cipher: c,
	}
}

// Run the crypto benchmark
func (b *BenchCrypto) Run() {

	b.encrypt()
	b.decrypt()

	// Sanity check
	if !bytes.Equal(b.input, b.out2.Bytes()) {
		log.Fatal("Encrypt/decrypt mismatch")
	}

	b.out1.Reset()
	b.out2.Reset()
}

func (b *BenchCrypto) encrypt() {

	// Create a new 3DES/CTR encoder
	var iv [des.BlockSize]byte
	ctr := cipher.NewCTR(b.cipher, iv[:])

	// Encrypting requires a temporary buffer
	w := &CryptoStreamWriter{S: ctr, W: &b.out1, B: b.buf}

	// Copy the input to the output buffer, encrypting as we go.
	rinput := bytes.NewReader(b.input)
	if _, err := io.Copy(w, rinput); err != nil {
		panic(err)
	}

	// Avoid reallocation of temporary buffer
	b.buf = w.B
	w.Close()
}

func (b *BenchCrypto) decrypt() {

	// Create a new 3DES/CTR decoder
	var iv [des.BlockSize]byte
	ctr := cipher.NewCTR(b.cipher, iv[:])

	// Decrypting does not require any temporary buffer
	r := &cipher.StreamReader{S: ctr, R: &b.out1}

	// Copy the input to the output buffer, decrypting as we go.
	if _, err := io.Copy(&b.out2, r); err != nil {
		panic(err)
	}
}

// CryptoStreamWriter is a cipher writer which keeps its internal buffer
type CryptoStreamWriter struct {
	S cipher.Stream
	W io.Writer
	B []byte
}

func (w *CryptoStreamWriter) Write(src []byte) (n int, err error) {
	if w.B == nil || len(w.B) < len(src) {
		w.B = make([]byte, len(src))
	}
	w.S.XORKeyStream(w.B, src)
	n, err = w.W.Write(w.B)
	if n != len(src) && err == nil {
		err = io.ErrShortWrite
	}
	return
}

func (w *CryptoStreamWriter) Close() error {
	if c, ok := w.W.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
