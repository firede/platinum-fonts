package bdf

import (
	"image"
	"strings"
	"testing"
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
