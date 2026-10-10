// Package labels renders label images for 12 mm TZe tape at 180 dpi.
//
// An image is laid out with x along the tape (label length) and y across it.
// Its height is the printable band of the tape (70 dots on 12 mm); the
// printer adds a feed margin (2 mm minimum) before and after each label.
package labels

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/datamatrix"
	"github.com/boombuler/barcode/qr"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type Symbology string

const (
	// DataMatrix encodes only the code; smallest label, scanned in the app.
	DataMatrix Symbology = "datamatrix"
	// QR encodes the full URL, so any phone camera opens the page.
	QR Symbology = "qr"
	// Code128 is a 1D barcode; long labels, for shelves and big bins.
	Code128 Symbology = "code128"
)

func ParseSymbology(s string) (Symbology, error) {
	switch Symbology(s) {
	case DataMatrix, QR, Code128:
		return Symbology(s), nil
	}
	return "", fmt.Errorf("unknown symbology %q", s)
}

const (
	// BandDots is the printable height of 12 mm tape.
	BandDots = 70
	// TapeDots is the full tape width, for previews.
	TapeDots = 85
	// MarginDots is the feed margin added by the printer on each side.
	MarginDots = 14

	gap = 3
)

var face = basicfont.Face7x13

// Spec is what goes on one label.
type Spec struct {
	Code      string    // e.g. "A107"
	Caption   string    // optional extra text, e.g. the item name
	Symbology Symbology //
	URL       string    // payload for QR codes; defaults to the code
}

// Render draws the label. The result is BandDots high.
func Render(s Spec) (*image.Gray, error) {
	sym, err := symbol(s)
	if err != nil {
		return nil, err
	}
	codeW := textWidth(s.Code)
	lineH := face.Height

	// Code text goes under the symbol if it fits, otherwise to its right.
	below := sym.Bounds().Dy()+gap+lineH <= BandDots
	var captionLines []string
	if s.Caption != "" {
		captionLines = wrap(s.Caption, 18, BandDots/lineH-boolInt(!below))
	}

	colW := sym.Bounds().Dx()
	if below {
		colW = max(colW, codeW)
	}
	w := colW
	var textX int
	var textLines []string
	if !below {
		textLines = append(textLines, s.Code)
	}
	textLines = append(textLines, captionLines...)
	if len(textLines) > 0 {
		// Wide enough to act as the symbol's quiet zone.
		textX = colW + 12
		tw := 0
		for _, l := range textLines {
			tw = max(tw, textWidth(l))
		}
		w = textX + tw
	}

	img := image.NewGray(image.Rect(0, 0, w, BandDots))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

	symH := sym.Bounds().Dy()
	blockH := symH
	if below {
		blockH += gap + lineH
	}
	top := (BandDots - blockH) / 2
	symX := (colW - sym.Bounds().Dx()) / 2
	draw.Draw(img, sym.Bounds().Add(image.Pt(symX, top)), sym, image.Point{}, draw.Src)
	if below {
		drawText(img, (colW-codeW)/2, top+symH+gap, s.Code)
	}
	if len(textLines) > 0 {
		ty := (BandDots - len(textLines)*lineH) / 2
		for i, l := range textLines {
			drawText(img, textX, ty+i*lineH, l)
		}
	}
	return img, nil
}

// symbol encodes and scales the code; quiet zones come from the tape edge
// and the feed margins.
func symbol(s Spec) (image.Image, error) {
	var (
		bc    barcode.Barcode
		err   error
		scale int
		h     int // 1D bar height
	)
	switch s.Symbology {
	case DataMatrix:
		bc, err = datamatrix.Encode(s.Code)
		scale = 3
	case QR:
		payload := s.URL
		if payload == "" {
			payload = s.Code
		}
		// Upper case keeps the QR in alphanumeric mode: a smaller symbol.
		bc, err = qr.Encode(strings.ToUpper(payload), qr.L, qr.Auto)
		scale = 2
	case Code128:
		bc, err = code128.Encode(s.Code)
		scale, h = 2, BandDots-gap-face.Height
	default:
		return nil, fmt.Errorf("unknown symbology %q", s.Symbology)
	}
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", s.Symbology, err)
	}
	b := bc.Bounds()
	if h == 0 {
		h = b.Dy() * scale
	}
	if h > BandDots {
		// Fall back to the smallest scale that still fits.
		scale = BandDots / b.Dy()
		if scale < 1 {
			return nil, fmt.Errorf("%s symbol for %q is too big for the tape", s.Symbology, s.Code)
		}
		h = b.Dy() * scale
	}
	out := image.NewGray(image.Rect(0, 0, b.Dx()*scale, h))
	is1D := s.Symbology == Code128
	for y := 0; y < h; y++ {
		sy := b.Min.Y + y/scale
		if is1D {
			sy = b.Min.Y
		}
		for x := 0; x < out.Bounds().Dx(); x++ {
			c := color.GrayModel.Convert(bc.At(b.Min.X+x/scale, sy)).(color.Gray)
			out.Pix[y*out.Stride+x] = c.Y
		}
	}
	return out, nil
}

// Preview draws the label as it will come out: on a piece of tape with the
// feed margins and the unprinted tape edges.
func Preview(label image.Image) *image.Gray {
	w := label.Bounds().Dx() + 2*MarginDots
	img := image.NewGray(image.Rect(0, 0, w, TapeDots))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(img, label.Bounds().Add(image.Pt(MarginDots, (TapeDots-BandDots)/2)), label, label.Bounds().Min, draw.Src)
	return img
}

// LengthMM is the length of tape a label uses, including feed margins.
func LengthMM(label image.Image) float64 {
	return float64(label.Bounds().Dx()+2*MarginDots) * 25.4 / 180
}

func textWidth(s string) int {
	return font.MeasureString(face, s).Ceil()
}

func drawText(img draw.Image, x, y int, s string) {
	d := font.Drawer{
		Dst:  img,
		Src:  image.Black,
		Face: face,
		Dot:  fixed.P(x, y+face.Ascent),
	}
	d.DrawString(s)
}

// wrap breaks s into at most maxLines lines of up to width characters,
// ellipsizing the last line if needed.
func wrap(s string, width, maxLines int) []string {
	if maxLines <= 0 {
		return nil
	}
	var lines []string
	var cur string
	for _, w := range strings.Fields(s) {
		for len(w) > width {
			if cur != "" {
				lines, cur = append(lines, cur), ""
			}
			lines, w = append(lines, w[:width]), w[width:]
		}
		switch {
		case cur == "":
			cur = w
		case len(cur)+1+len(w) <= width:
			cur += " " + w
		default:
			lines, cur = append(lines, cur), w
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		last := lines[maxLines-1]
		if len(last) > width-3 {
			last = last[:width-3]
		}
		lines[maxLines-1] = last + "..." // the bitmap font is ASCII only
	}
	return lines
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
