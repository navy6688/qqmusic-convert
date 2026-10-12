// Generate reproducible code-native logo PNG/ICO assets without a raster editor.
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

const sourceSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256">
<defs><linearGradient id="bg" x2="1" y2="1"><stop stop-color="#193b5a"/><stop offset="1" stop-color="#0c213a"/></linearGradient></defs>
<rect x="8" y="8" width="240" height="240" rx="53" fill="url(#bg)"/>
<path d="M188 62 A84 84 0 1 0 207 174" fill="none" stroke="#42d4af" stroke-width="14" stroke-linecap="round"/>
<path d="M190 156 L207 180 L226 158" fill="none" stroke="#42d4af" stroke-width="13" stroke-linecap="round" stroke-linejoin="round"/>
<path d="M112 161 V91 L166 78 V148 M112 108 L166 95" fill="none" stroke="white" stroke-width="12" stroke-linecap="round" stroke-linejoin="round"/>
<ellipse cx="100" cy="164" rx="18" ry="13" fill="white"/><ellipse cx="154" cy="151" rx="18" ry="13" fill="white"/>
</svg>`

func main() {
	output := flag.String("output", "internal/desktop/assets", "logo asset directory")
	flag.Parse()
	if err := generate(*output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(directory string) error {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(directory, "logo.svg"), []byte(sourceSVG), 0644); err != nil {
		return err
	}
	var images [][]byte
	sizes := []int{16, 24, 32, 48, 64, 128, 256}
	for _, size := range sizes {
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, render(size)); err != nil {
			return err
		}
		images = append(images, encoded.Bytes())
		if size == 256 {
			if err := os.WriteFile(filepath.Join(directory, "logo.png"), encoded.Bytes(), 0644); err != nil {
				return err
			}
		}
	}
	var ico bytes.Buffer
	for _, value := range []uint16{0, 1, uint16(len(sizes))} {
		if err := binary.Write(&ico, binary.LittleEndian, value); err != nil {
			return err
		}
	}
	offset := 6 + 16*len(sizes)
	for index, size := range sizes {
		dimension := byte(size)
		if size == 256 {
			dimension = 0
		}
		ico.Write([]byte{dimension, dimension, 0, 0})
		for _, value := range []uint16{1, 32} {
			if err := binary.Write(&ico, binary.LittleEndian, value); err != nil {
				return err
			}
		}
		for _, value := range []uint32{uint32(len(images[index])), uint32(offset)} {
			if err := binary.Write(&ico, binary.LittleEndian, value); err != nil {
				return err
			}
		}
		offset += len(images[index])
	}
	for _, encoded := range images {
		ico.Write(encoded)
	}
	return os.WriteFile(filepath.Join(directory, "logo.ico"), ico.Bytes(), 0644)
}

func render(size int) *image.NRGBA {
	icon := image.NewNRGBA(image.Rect(0, 0, size, size))
	const samples = 4
	for row := range size {
		for column := range size {
			var sums [4]int
			for sy := range samples {
				for sx := range samples {
					x := (float64(column) + (float64(sx)+0.5)/samples) * 256 / float64(size)
					y := (float64(row) + (float64(sy)+0.5)/samples) * 256 / float64(size)
					pixel := at(x, y)
					sums[0] += int(pixel.R) * int(pixel.A)
					sums[1] += int(pixel.G) * int(pixel.A)
					sums[2] += int(pixel.B) * int(pixel.A)
					sums[3] += int(pixel.A)
				}
			}
			if sums[3] == 0 {
				continue
			}
			icon.SetNRGBA(column, row, color.NRGBA{R: uint8(sums[0] / sums[3]), G: uint8(sums[1] / sums[3]), B: uint8(sums[2] / sums[3]), A: uint8(sums[3] / (samples * samples))})
		}
	}
	return icon
}

func at(x, y float64) color.NRGBA {
	dx, dy := math.Max(math.Abs(x-128)-67, 0), math.Max(math.Abs(y-128)-67, 0)
	if dx*dx+dy*dy > 53*53 {
		return color.NRGBA{}
	}
	fraction := (x + y) / 512
	pixel := color.NRGBA{R: uint8(25 - 13*fraction), G: uint8(59 - 26*fraction), B: uint8(90 - 32*fraction), A: 255}
	angle := math.Atan2(y-128, x-128) * 180 / math.Pi
	radius := math.Hypot(x-128, y-128)
	if math.Abs(radius-84) <= 7 && (angle >= 30 || angle <= -45) || segmentDistance(x, y, 190, 156, 207, 180) <= 6.5 || segmentDistance(x, y, 207, 180, 226, 158) <= 6.5 {
		pixel = color.NRGBA{R: 66, G: 212, B: 175, A: 255}
	}
	if segmentDistance(x, y, 112, 161, 112, 91) <= 6 || segmentDistance(x, y, 112, 91, 166, 78) <= 6 || segmentDistance(x, y, 166, 78, 166, 148) <= 6 || segmentDistance(x, y, 112, 108, 166, 95) <= 6 || ellipse(x, y, 100, 164, 18, 13) || ellipse(x, y, 154, 151, 18, 13) {
		pixel = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	}
	return pixel
}

func segmentDistance(x, y, x1, y1, x2, y2 float64) float64 {
	vx, vy := x2-x1, y2-y1
	projection := math.Min(1, math.Max(0, ((x-x1)*vx+(y-y1)*vy)/(vx*vx+vy*vy)))
	return math.Hypot(x-x1-projection*vx, y-y1-projection*vy)
}
func ellipse(x, y, cx, cy, rx, ry float64) bool {
	return math.Pow((x-cx)/rx, 2)+math.Pow((y-cy)/ry, 2) <= 1
}
