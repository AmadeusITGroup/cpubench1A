package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"os"
	"strings"
)

const IMG_W = 128
const IMG_H = 128
const IMG_DEBUG = false

// ImageBuffer is a normal buffer with a dummy Flush to avoid extra allocations with jpeg.Encode
type ImageBuffer struct {
	bytes.Buffer
}

// Flush does nothing
func (ib *ImageBuffer) Flush() error {
	return nil
}

// BenchImage is an image composition and encoding benchmark
type BenchImage struct {
	img *image.RGBA
	wil *image.RGBA
	buf ImageBuffer
}

// NewBenchImage allocates a new benchmark object
func NewBenchImage() *BenchImage {

	// Read Wilson image
	wilson, err := jpeg.Decode(base64.NewDecoder(base64.StdEncoding, strings.NewReader(base64Wilson)))
	if err != nil {
		log.Fatal("Cannot decode image - ", err)
	}

	// Convert it to RGBA
	wil := image.NewRGBA(wilson.Bounds())
	draw.Draw(wil, wil.Bounds(), wilson, image.Point{}, draw.Src)

	return &BenchImage{
		img: image.NewRGBA(image.Rect(0, 0, IMG_W, IMG_H)),
		wil: wil,
	}
}

// Run does compose an image and generate jpeg data from it
func (b *BenchImage) Run() {

	b.chessboard()
	b.compose()
	b.generate()
	if IMG_DEBUG {
		b.dump("debug_image.jpg")
	}
}

// chessboard composes a chessboard image
func (b *BenchImage) chessboard() {

	colors := []*image.Uniform{
		{C: color.RGBA{0, 100, 0, 255}},
		{C: color.RGBA{50, 205, 50, 255}},
	}
	ic := 0
	sb := IMG_H / 8
	lx := (IMG_W - IMG_H) / 2

	for x := 0; x < 8; x++ {
		ly := 0
		for y := 0; y < 8; y++ {
			draw.Draw(b.img, image.Rect(lx, ly, lx+sb, ly+sb), colors[ic], image.Point{}, draw.Src)
			ly += sb
			ic = 1 - ic
		}
		lx += sb
		ic = 1 - ic
	}
}

// compose merges the Wilson image and adds a small alpha transparency window
func (b *BenchImage) compose() {

	w, h := b.wil.Bounds().Dx(), b.wil.Bounds().Dy()
	x, y := (IMG_W-w)/2, (IMG_H-h)/2
	r := image.Rect(x, y, x+w, y+h)
	draw.Draw(b.img, r, b.wil, image.Point{}, draw.Src)
	draw.DrawMask(b.img, image.Rect(x-16, y-16, x+32, y+32), image.NewUniform(color.Black), image.Point{}, image.NewUniform(color.Alpha{128}), image.Point{}, draw.Over)
}

// generate writes jpeg data in memory
func (b *BenchImage) generate() error {

	if err := jpeg.Encode(&b.buf, b.img, &jpeg.Options{Quality: 75}); err != nil {
		return nil
	}
	b.buf.Reset()

	return nil
}

// dump only used for debugging purpose
func (b *BenchImage) dump(path string) error {

	f, err := os.Create(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	if err = jpeg.Encode(f, b.img, &jpeg.Options{Quality: 85}); err != nil {
		return nil
	}
	return nil
}

const base64Wilson = `
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
`
