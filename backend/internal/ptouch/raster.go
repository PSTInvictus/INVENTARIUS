// Package ptouch drives Brother PT-E550W / PT-P750W label printers using the
// raster protocol in docs/brother-pt-e550w-raster-reference.pdf
// ("Raster Command Reference PT-E550W/P750W/P710BT", v1.02).
package ptouch

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
)

const (
	headPins     = 128
	bytesPerLine = headPins / 8

	// MinMarginDots is the smallest feed margin the printer accepts (2 mm at 180 dpi).
	MinMarginDots = 14
	// DPI is the resolution along and across the tape in normal mode.
	DPI = 180
)

// Tape describes the printable band of one TZe tape width (spec §2.3.5).
type Tape struct {
	WidthMM    int
	LeftPins   int // unused pins before the print area
	PrintPins  int // height of the printable band, in dots
	MediaWidth byte
}

var tapes = map[int]Tape{
	4:  {WidthMM: 4, LeftPins: 52, PrintPins: 24, MediaWidth: 4}, // 3.5 mm reports as 4
	6:  {WidthMM: 6, LeftPins: 48, PrintPins: 32, MediaWidth: 6},
	9:  {WidthMM: 9, LeftPins: 39, PrintPins: 50, MediaWidth: 9},
	12: {WidthMM: 12, LeftPins: 29, PrintPins: 70, MediaWidth: 12},
	18: {WidthMM: 18, LeftPins: 8, PrintPins: 112, MediaWidth: 18},
	24: {WidthMM: 24, LeftPins: 0, PrintPins: 128, MediaWidth: 24},
}

// TapeFor returns the layout for a tape width in mm as reported by the printer.
func TapeFor(widthMM int) (Tape, bool) {
	t, ok := tapes[widthMM]
	return t, ok
}

// JobOptions control cutting and margins for one print job.
type JobOptions struct {
	Tape Tape
	// MediaType from the status reply (01h laminated, 03h non-laminated...).
	MediaType byte
	// MarginDots is fed before and after each label; at least MinMarginDots.
	MarginDots int
	// HalfCut half-cuts between labels (the backing stays whole, so a batch
	// comes out as one peelable strip) and fully cuts only after the last one.
	HalfCut bool
}

// Encode builds a complete print job: one page per label image. Each image
// is laid out with x along the tape (feed direction, one raster line per
// column) and y across it; its height must not exceed Tape.PrintPins, and
// it is centred in the printable band. Dark pixels print.
func Encode(labels []image.Image, o JobOptions) ([]byte, error) {
	if len(labels) == 0 {
		return nil, fmt.Errorf("ptouch: no labels")
	}
	if o.Tape.PrintPins == 0 {
		return nil, fmt.Errorf("ptouch: unknown tape")
	}
	if o.MarginDots < MinMarginDots {
		o.MarginDots = MinMarginDots
	}
	var b bytes.Buffer

	// (1) Initialization: invalidate, then initialize.
	b.Write(make([]byte, 100))
	b.Write([]byte{0x1b, 0x40})

	for i, img := range labels {
		r := img.Bounds()
		if r.Dy() > o.Tape.PrintPins {
			return nil, fmt.Errorf("ptouch: label %d is %d dots high, tape prints %d", i, r.Dy(), o.Tape.PrintPins)
		}
		if r.Dx() == 0 {
			return nil, fmt.Errorf("ptouch: label %d is empty", i)
		}
		first, last := i == 0, i == len(labels)-1

		// (2) Control codes, repeated for each page.
		b.Write([]byte{0x1b, 0x69, 0x61, 0x01}) // switch to raster mode

		// Print information: valid flags = recover | width. n9 = 0 on the
		// first page, 1 on the others.
		var lines [4]byte
		binary.LittleEndian.PutUint32(lines[:], uint32(r.Dx()))
		page := byte(1)
		if first {
			page = 0
		}
		b.Write([]byte{0x1b, 0x69, 0x7a, 0x84, o.MediaType, o.Tape.MediaWidth, 0x00})
		b.Write(lines[:])
		b.Write([]byte{page, 0x00})

		b.Write([]byte{0x1b, 0x69, 0x4d, 0x40}) // various mode: auto cut
		cutEvery := 1
		if o.HalfCut {
			cutEvery = min(len(labels), 99)
		}
		b.Write([]byte{0x1b, 0x69, 0x41, byte(cutEvery)}) // full cut every N labels

		adv := byte(0x08) // no chain printing: feed and cut after the last label
		if o.HalfCut {
			adv |= 0x04
		}
		b.Write([]byte{0x1b, 0x69, 0x4b, adv})

		b.Write([]byte{0x1b, 0x69, 0x64, byte(o.MarginDots), byte(o.MarginDots >> 8)})
		b.Write([]byte{0x4d, 0x02}) // TIFF (PackBits) compression

		// (3) Raster data.
		yOff := (o.Tape.PrintPins - r.Dy()) / 2
		for x := r.Min.X; x < r.Max.X; x++ {
			var line [bytesPerLine]byte
			empty := true
			for y := r.Min.Y; y < r.Max.Y; y++ {
				if !dark(img, x, y) {
					continue
				}
				// The top of the image goes to the highest pin, matching the
				// reading direction as the label leaves the printer.
				pin := o.Tape.LeftPins + yOff + (r.Max.Y - 1 - y)
				line[bytesPerLine-1-pin/8] |= 1 << (pin % 8)
				empty = false
			}
			if empty {
				b.WriteByte(0x5a) // zero raster graphics
				continue
			}
			packed := PackBits(line[:])
			b.Write([]byte{0x47, byte(len(packed)), byte(len(packed) >> 8)})
			b.Write(packed)
		}

		// (4) Print command.
		if last {
			b.WriteByte(0x1a)
		} else {
			b.WriteByte(0x0c)
		}
	}
	return b.Bytes(), nil
}

func dark(img image.Image, x, y int) bool {
	r, g, bl, a := img.At(x, y).RGBA()
	if a < 0x8000 {
		return false
	}
	// Rec. 601 luma on 16-bit channels.
	return (299*r+587*g+114*bl)/1000 < 0x8000
}

// PackBits compresses data with the TIFF PackBits scheme used by the "G"
// command: runs of 2+ equal bytes become (1-n, b), other bytes go out as
// literal blocks prefixed with (n-1). Blocks hold at most 128 bytes.
func PackBits(data []byte) []byte {
	var out []byte
	for i := 0; i < len(data); {
		run := 1
		for i+run < len(data) && data[i+run] == data[i] && run < 128 {
			run++
		}
		if run >= 2 {
			out = append(out, byte(1-run), data[i])
			i += run
			continue
		}
		// Literal block: up to the start of the next run of 2+.
		j := i + 1
		for j < len(data) && j-i < 128 && !(j+1 < len(data) && data[j] == data[j+1]) {
			j++
		}
		out = append(out, byte(j-i-1))
		out = append(out, data[i:j]...)
		i = j
	}
	return out
}

// UnpackBits reverses PackBits; used by tests and the fake printer.
func UnpackBits(data []byte) ([]byte, error) {
	var out []byte
	for i := 0; i < len(data); {
		n := int8(data[i])
		i++
		switch {
		case n >= 0:
			if i+int(n)+1 > len(data) {
				return nil, fmt.Errorf("packbits: literal overruns input")
			}
			out = append(out, data[i:i+int(n)+1]...)
			i += int(n) + 1
		case n != -128:
			if i >= len(data) {
				return nil, fmt.Errorf("packbits: run overruns input")
			}
			out = append(out, bytes.Repeat([]byte{data[i]}, 1-int(n))...)
			i++
		}
	}
	return out, nil
}
