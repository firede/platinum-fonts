package bdf

import (
	"bytes"
	"image"
	"strings"
	"testing"

	"github.com/firede/platinum-fonts/bitmap"
)

const sample = `STARTFONT 2.1
FONT -test-
SIZE 12 75 75
FONTBOUNDINGBOX 12 12 0 -2
STARTPROPERTIES 2
FONT_ASCENT 10
FONT_DESCENT 2
ENDPROPERTIES
CHARS 2
STARTCHAR uni4E2D
ENCODING 20013
SWIDTH 1000 0
DWIDTH 12 0
BBX 3 4 1 -1
BITMAP
40
E0
A0
40
ENDCHAR
STARTCHAR unencoded
ENCODING -1
DWIDTH 6 0
BBX 1 1 0 0
BITMAP
80
ENDCHAR
ENDFONT
`

func TestParse(t *testing.T) {
	f, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if f.Ascent != 10 || f.Descent != 2 || len(f.Glyphs) != 1 {
		t.Fatalf("ascent %d, descent %d, %d glyphs", f.Ascent, f.Descent, len(f.Glyphs))
	}
	g := f.Glyphs['中']
	// 4 rows whose bottom is a row under the baseline, from x = 1.
	if g.Advance != 12 || g.Bounds != image.Rect(1, -3, 4, 1) {
		t.Fatalf("advance %d, bounds %v", g.Advance, g.Bounds)
	}
	var b strings.Builder
	for y := -3; y < 1; y++ {
		for x := 1; x < 4; x++ {
			if g.At(x, y) {
				b.WriteByte('#')
			} else {
				b.WriteByte('.')
			}
		}
		b.WriteByte('/')
	}
	if b.String() != ".#./###/#.#/.#./" {
		t.Errorf("drawing %s", b.String())
	}
}

func TestParseErrors(t *testing.T) {
	for name, src := range map[string]string{
		"rows":       strings.Replace(sample, "40\nENDCHAR", "ENDCHAR", 1),
		"hex":        strings.Replace(sample, "E0", "Z0", 1),
		"unfinished": strings.TrimSuffix(sample, "ENDCHAR\nENDFONT\n"),
		"bbx":        strings.Replace(sample, "BBX 3 4 1 -1", "BBX 3 4 1", 1),
	} {
		if _, err := Parse([]byte(src)); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestWriteRoundTrip(t *testing.T) {
	in, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	wide := bitmap.NewGlyph(18, image.Rect(0, -9, 17, 2), func(x, y int) bool { return (x+y)%3 == 0 })
	in.Glyphs[0x20000] = wide
	in.Glyphs[' '] = bitmap.Glyph{Advance: 4}
	notdef := bitmap.NewGlyph(7, image.Rect(1, -8, 6, 0), func(x, y int) bool { return x == 1 || y == -1 })
	var out bytes.Buffer
	h := Header{Name: "-test-", PixelSize: 12, Ascent: 11, Descent: 3, Properties: map[string]string{"FAMILY_NAME": `"Test"`}}
	named := map[string]bitmap.Glyph{"lig.003D_003D.0": bitmap.NewGlyph(6, image.Rect(0, -5, 6, -2), func(x, y int) bool { return y != -4 })}
	if err := Write(&out, h, &notdef, named, in.Glyphs); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{"STARTCHAR .notdef\nENCODING -1\n", "FAMILY_NAME \"Test\"\n", "STARTCHAR u20000\n", "CHARS 5\n", "STARTCHAR lig.003D_003D.0\nENCODING -1\n"} {
		if !strings.Contains(text, want) {
			t.Errorf("no %q in\n%s", want, text)
		}
	}
	back, err := Parse(out.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if back.Ascent != 11 || back.Descent != 3 || len(back.Glyphs) != len(in.Glyphs) {
		t.Fatalf("ascent %d, descent %d, %d glyphs", back.Ascent, back.Descent, len(back.Glyphs))
	}
	for r, g := range in.Glyphs {
		b := back.Glyphs[r]
		if b.Advance != g.Advance || b.Bounds != g.Bounds {
			t.Fatalf("%U: advance %d in %v, wrote %d in %v", r, b.Advance, b.Bounds, g.Advance, g.Bounds)
		}
		for y := g.Bounds.Min.Y; y < g.Bounds.Max.Y; y++ {
			for x := g.Bounds.Min.X; x < g.Bounds.Max.X; x++ {
				if b.At(x, y) != g.At(x, y) {
					t.Fatalf("%U differs at %d, %d", r, x, y)
				}
			}
		}
	}
}
