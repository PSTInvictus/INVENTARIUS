package ptouch

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPackBitsSpecExample(t *testing.T) {
	// Example from the "Select compression mode" section.
	in := append(make([]byte, 20), 0x22, 0x22, 0x23, 0xba, 0xbf, 0xa2, 0x22, 0x2b)
	want := []byte{0xed, 0x00, 0xff, 0x22, 0x05, 0x23, 0xba, 0xbf, 0xa2, 0x22, 0x2b}
	if got := PackBits(in); !bytes.Equal(got, want) {
		t.Fatalf("PackBits = % x, want % x", got, want)
	}
}

func TestPackBitsRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 2000; i++ {
		in := make([]byte, r.Intn(300))
		for j := range in {
			if r.Intn(3) == 0 {
				in[j] = byte(r.Intn(256))
			} else if j > 0 {
				in[j] = in[j-1]
			}
		}
		out, err := UnpackBits(PackBits(in))
		if err != nil || !bytes.Equal(out, in) {
			t.Fatalf("round trip failed for % x: got % x, %v", in, out, err)
		}
	}
}

// decodedPage is what the fake printer reconstructs from a job.
type decodedPage struct {
	widthMM, lines int
	margin         int
	adv, mode      byte
	cutEvery       byte
	pins           [][headPins]bool // [line][pin]
	last           bool
}

// decodeJob parses a job the way the printer would, failing on anything
// that does not follow the documented command sequence.
func decodeJob(t *testing.T, b []byte) []decodedPage {
	t.Helper()
	if len(b) < 102 || !bytes.Equal(b[:100], make([]byte, 100)) || b[100] != 0x1b || b[101] != 0x40 {
		t.Fatalf("job must start with 100-byte invalidate and ESC @")
	}
	b = b[102:]
	var pages []decodedPage
	var cur *decodedPage
	expect := func(prefix ...byte) {
		t.Helper()
		if !bytes.HasPrefix(b, prefix) {
			t.Fatalf("expected % x, got % x", prefix, b[:min(len(b), len(prefix)+4)])
		}
		b = b[len(prefix):]
	}
	for len(b) > 0 {
		if cur == nil {
			pages = append(pages, decodedPage{})
			cur = &pages[len(pages)-1]
			expect(0x1b, 0x69, 0x61, 0x01)
			expect(0x1b, 0x69, 0x7a, 0x84)
			cur.widthMM = int(b[1])
			cur.lines = int(binary.LittleEndian.Uint32(b[3:7]))
			if wantPage := byte(min(len(pages)-1, 1)); b[7] != wantPage {
				t.Fatalf("page %d: starting-page flag %d", len(pages), b[7])
			}
			b = b[9:]
			expect(0x1b, 0x69, 0x4d)
			cur.mode, b = b[0], b[1:]
			expect(0x1b, 0x69, 0x41)
			cur.cutEvery, b = b[0], b[1:]
			expect(0x1b, 0x69, 0x4b)
			cur.adv, b = b[0], b[1:]
			expect(0x1b, 0x69, 0x64)
			cur.margin, b = int(b[0])|int(b[1])<<8, b[2:]
			expect(0x4d, 0x02)
			continue
		}
		switch b[0] {
		case 0x5a:
			cur.pins = append(cur.pins, [headPins]bool{})
			b = b[1:]
		case 0x47:
			n := int(b[1]) | int(b[2])<<8
			line, err := UnpackBits(b[3 : 3+n])
			if err != nil || len(line) != bytesPerLine {
				t.Fatalf("bad raster line % x: %v", b[3:3+n], err)
			}
			var pins [headPins]bool
			for p := 0; p < headPins; p++ {
				pins[p] = line[bytesPerLine-1-p/8]&(1<<(p%8)) != 0
			}
			cur.pins = append(cur.pins, pins)
			b = b[3+n:]
		case 0x0c, 0x1a:
			cur.last = b[0] == 0x1a
			if len(cur.pins) != cur.lines {
				t.Fatalf("page declared %d raster lines, sent %d", cur.lines, len(cur.pins))
			}
			cur = nil
			b = b[1:]
		default:
			t.Fatalf("unexpected byte %#x in raster data", b[0])
		}
	}
	if cur != nil {
		t.Fatalf("job ended without a print command")
	}
	return pages
}

func testImage(w, h int, seed int64) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, w, h))
	r := rand.New(rand.NewSource(seed))
	for i := range img.Pix {
		if r.Intn(2) == 0 {
			img.Pix[i] = 0xff
		}
	}
	return img
}

func TestEncodeRoundTrip(t *testing.T) {
	tape, _ := TapeFor(12)
	imgs := []image.Image{testImage(30, 58, 1), testImage(50, 70, 2), image.NewGray(image.Rect(0, 0, 5, 70))}
	for i := range imgs[2].(*image.Gray).Pix {
		imgs[2].(*image.Gray).Pix[i] = 0xff // blank label: all zero-raster lines
	}
	job, err := Encode(imgs, JobOptions{Tape: tape, MediaType: 0x01, HalfCut: true})
	if err != nil {
		t.Fatal(err)
	}
	pages := decodeJob(t, job)
	if len(pages) != 3 {
		t.Fatalf("got %d pages", len(pages))
	}
	for i, p := range pages {
		img := imgs[i]
		w, h := img.Bounds().Dx(), img.Bounds().Dy()
		if p.widthMM != 12 || p.lines != w || p.margin != MinMarginDots {
			t.Errorf("page %d header: %+v", i, p)
		}
		if p.mode != 0x40 || p.adv != 0x0c || p.cutEvery != 3 {
			t.Errorf("page %d cut settings: mode %#x adv %#x every %d", i, p.mode, p.adv, p.cutEvery)
		}
		if p.last != (i == 2) {
			t.Errorf("page %d last=%v", i, p.last)
		}
		yOff := (tape.PrintPins - h) / 2
		for x := 0; x < w; x++ {
			for pin := 0; pin < headPins; pin++ {
				want := false
				if y := tape.LeftPins + yOff + h - 1 - pin; pin >= tape.LeftPins+yOff && y >= 0 && y < h {
					want = img.(*image.Gray).GrayAt(x, y).Y == 0
				}
				if p.pins[x][pin] != want {
					t.Fatalf("page %d line %d pin %d = %v, want %v", i, x, pin, p.pins[x][pin], want)
				}
			}
		}
	}
}

func TestEncodeSingleLabelFullCut(t *testing.T) {
	tape, _ := TapeFor(12)
	job, err := Encode([]image.Image{testImage(10, 10, 3)}, JobOptions{Tape: tape, MarginDots: 3})
	if err != nil {
		t.Fatal(err)
	}
	p := decodeJob(t, job)[0]
	if p.adv != 0x08 || p.cutEvery != 1 || p.margin != MinMarginDots {
		t.Fatalf("single label: adv %#x every %d margin %d", p.adv, p.cutEvery, p.margin)
	}
}

func TestEncodeRejectsTallLabel(t *testing.T) {
	tape, _ := TapeFor(12)
	if _, err := Encode([]image.Image{testImage(10, 71, 4)}, JobOptions{Tape: tape}); err == nil {
		t.Fatal("expected error for a label taller than the print area")
	}
}

func statusPacket(typ, err1, err2, width, media byte) []byte {
	p := make([]byte, 32)
	p[0], p[1], p[2], p[3], p[4], p[5] = 0x80, 0x20, 'B', '0', ModelPTE550W, '0'
	p[8], p[9], p[10], p[11], p[18] = err1, err2, width, media, typ
	return p
}

// fakePrinter answers a status request and reports each page as printed.
type fakePrinter struct {
	width, err1 byte
	silent      bool // never sends anything (no read support)
	failAfter   int  // report an error after this many pages (0 = never)
	written     bytes.Buffer
	replies     [][]byte
}

func (f *fakePrinter) Write(p []byte) (int, error) {
	f.written.Write(p)
	if f.silent {
		return len(p), nil
	}
	if bytes.HasSuffix(p, []byte{0x1b, 0x69, 0x53}) {
		f.replies = append(f.replies, statusPacket(StatusReply, f.err1, 0, f.width, 0x01))
		return len(p), nil
	}
	pages := bytes.Count(p, []byte{0x1b, 0x69, 0x61, 0x01})
	for i := 1; i <= pages; i++ {
		if f.failAfter != 0 && i > f.failAfter {
			f.replies = append(f.replies, statusPacket(StatusError, 0, 0x10, f.width, 0x01))
			break
		}
		f.replies = append(f.replies, statusPacket(StatusPrinted, 0, 0, f.width, 0x01))
	}
	return len(p), nil
}

func (f *fakePrinter) ReadPacket(time.Time) ([]byte, error) {
	if len(f.replies) == 0 {
		return nil, os.ErrDeadlineExceeded
	}
	p := f.replies[0]
	f.replies = f.replies[1:]
	return p, nil
}

func (f *fakePrinter) Close() error { return nil }

func (f *fakePrinter) printer() *Printer {
	return &Printer{Dial: func() (Conn, error) { return f, nil }}
}

func TestPrint(t *testing.T) {
	labels := []image.Image{testImage(20, 50, 5), testImage(20, 50, 6)}
	tests := []struct {
		name      string
		fake      fakePrinter
		want      int
		errSubstr string
		confirmed bool
	}{
		{"ok", fakePrinter{width: 12}, 12, "", true},
		{"no tape", fakePrinter{width: 0, err1: 0x01}, 12, "no tape", false},
		{"wrong width", fakePrinter{width: 24}, 12, "24 mm tape loaded", false},
		{"error mid-job", fakePrinter{width: 12, failAfter: 1}, 12, "cover open", false},
		{"no reply", fakePrinter{silent: true}, 12, "did not answer", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := tc.fake.printer().Print(context.Background(), labels, tc.want)
			if tc.errSubstr == "" && err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if tc.errSubstr != "" && (err == nil || !strings.Contains(err.Error(), tc.errSubstr)) {
				t.Fatalf("error %v, want %q", err, tc.errSubstr)
			}
			if res.Confirmed != tc.confirmed {
				t.Fatalf("confirmed=%v", res.Confirmed)
			}
		})
	}
}

func TestStatusProblems(t *testing.T) {
	s, err := ParseStatus(statusPacket(StatusReply, 0x01|0x08, 0x10, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(s.Problems()); got != "[no tape weak batteries cover open]" {
		t.Fatalf("problems = %s", got)
	}
	if s.ModelName() != "PT-E550W" {
		t.Fatalf("model %s", s.ModelName())
	}
	if _, err := ParseStatus([]byte{1, 2, 3}); err == nil {
		t.Fatal("short packet accepted")
	}
}
