// Package bdf reads and writes fonts in the Glyph Bitmap Distribution
// Format (BDF) 2.1.
package bdf

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/firede/platinum-fonts/bitmap"
)

// Font is a parsed BDF font.
type Font struct {
	Ascent, Descent int
	Glyphs          map[rune]bitmap.Glyph
}

// Parse reads a BDF font. It keeps the glyphs of characters with a Unicode
// encoding, and reports what it cannot read with the line it is on.
func Parse(data []byte) (*Font, error) {
	f := &Font{Glyphs: map[rune]bitmap.Glyph{}}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	line := 0
	fail := func(format string, a ...any) error {
		return fmt.Errorf("bdf: line %d: %s", line, fmt.Sprintf(format, a...))
	}
	var (
		inChar, inBitmap bool
		enc, advance     int
		box              image.Rectangle
		rows             []uint64
		rowDigits        int
	)
	ints := func(fields []string, n int) ([]int, error) {
		if len(fields) < n+1 {
			return nil, fail("%s takes %d numbers", fields[0], n)
		}
		v := make([]int, n)
		for i := range v {
			var err error
			if v[i], err = strconv.Atoi(fields[i+1]); err != nil {
				return nil, fail("%s: %v", fields[0], err)
			}
		}
		return v, nil
	}
	for sc.Scan() {
		line++
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 {
			continue
		}
		if inBitmap && fields[0] != "ENDCHAR" {
			v, err := strconv.ParseUint(fields[0], 16, 64)
			if err != nil || len(fields[0]) > 16 {
				return nil, fail("a row of the bitmap is not hexadecimal")
			}
			rows = append(rows, v)
			rowDigits = len(fields[0])
			continue
		}
		switch fields[0] {
		case "FONT_ASCENT", "FONT_DESCENT":
			v, err := ints(fields, 1)
			if err != nil {
				return nil, err
			}
			if fields[0] == "FONT_ASCENT" {
				f.Ascent = v[0]
			} else {
				f.Descent = v[0]
			}
		case "STARTCHAR":
			inChar, enc, advance, box, rows = true, -1, 0, image.Rectangle{}, nil
		case "ENCODING":
			v, err := ints(fields, 1)
			if err != nil {
				return nil, err
			}
			enc = v[0]
		case "DWIDTH":
			v, err := ints(fields, 2)
			if err != nil {
				return nil, err
			}
			advance = v[0]
		case "BBX":
			v, err := ints(fields, 4)
			if err != nil {
				return nil, err
			}
			// BBX is w, h and the offset of its bottom left corner from the
			// dot, y up.
			w, h, x, y := v[0], v[1], v[2], v[3]
			box = image.Rect(x, -(y + h), x+w, -y)
		case "BITMAP":
			if !inChar {
				return nil, fail("BITMAP outside a character")
			}
			inBitmap = true
		case "ENDCHAR":
			if !inChar {
				return nil, fail("ENDCHAR outside a character")
			}
			if len(rows) != box.Dy() {
				return nil, fail("%d rows for a box %d tall", len(rows), box.Dy())
			}
			if rowDigits*4 < box.Dx() && box.Dy() > 0 {
				return nil, fail("rows of %d bits for a box %d wide", rowDigits*4, box.Dx())
			}
			if enc >= 0 {
				r, bits, top, left := rows, rowDigits*4, box.Min.Y, box.Min.X
				f.Glyphs[rune(enc)] = bitmap.NewGlyph(advance, box, func(x, y int) bool {
					return r[y-top]>>(bits-1-(x-left))&1 == 1
				})
			}
			inChar, inBitmap = false, false
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("bdf: %v", err)
	}
	if inChar {
		return nil, fail("the last character does not end")
	}
	return f, nil
}

// Header is what Write puts before the glyphs of a font.
type Header struct {
	Name            string            // the FONT line
	PixelSize       int               // SIZE and PIXEL_SIZE
	Ascent, Descent int               // FONT_ASCENT and FONT_DESCENT
	Properties      map[string]string // more properties, values written as they are
}

// Write writes a BDF font of glyphs, by character, sorted by character.
// notdef, if it has a box, is written first, as .notdef with no encoding.
func Write(w io.Writer, h Header, notdef *bitmap.Glyph, glyphs map[rune]bitmap.Glyph) error {
	runes := slices.Sorted(maps.Keys(glyphs))
	box := image.Rectangle{}
	all := func(fn func(bitmap.Glyph)) {
		if notdef != nil {
			fn(*notdef)
		}
		for _, r := range runes {
			fn(glyphs[r])
		}
	}
	all(func(g bitmap.Glyph) { box = box.Union(g.Bounds) })
	b := bufio.NewWriter(w)
	fmt.Fprintf(b, "STARTFONT 2.1\nFONT %s\nSIZE %d 75 75\n", h.Name, h.PixelSize)
	fmt.Fprintf(b, "FONTBOUNDINGBOX %d %d %d %d\n", box.Dx(), box.Dy(), box.Min.X, -box.Max.Y)
	keys := slices.Sorted(maps.Keys(h.Properties))
	fmt.Fprintf(b, "STARTPROPERTIES %d\n", len(keys)+3)
	fmt.Fprintf(b, "PIXEL_SIZE %d\nFONT_ASCENT %d\nFONT_DESCENT %d\n", h.PixelSize, h.Ascent, h.Descent)
	for _, k := range keys {
		fmt.Fprintf(b, "%s %s\n", k, h.Properties[k])
	}
	n := len(runes)
	if notdef != nil {
		n++
	}
	fmt.Fprintf(b, "ENDPROPERTIES\nCHARS %d\n", n)
	char := func(name string, enc int, g bitmap.Glyph) {
		gb := g.Bounds
		fmt.Fprintf(b, "STARTCHAR %s\nENCODING %d\n", name, enc)
		fmt.Fprintf(b, "SWIDTH %d 0\nDWIDTH %d 0\n", g.Advance*1000/h.PixelSize, g.Advance)
		fmt.Fprintf(b, "BBX %d %d %d %d\nBITMAP\n", gb.Dx(), gb.Dy(), gb.Min.X, -gb.Max.Y)
		digits := (gb.Dx() + 7) / 8 * 2
		for y := gb.Min.Y; y < gb.Max.Y; y++ {
			var row uint64
			for x := gb.Min.X; x < gb.Max.X; x++ {
				row <<= 1
				if g.At(x, y) {
					row |= 1
				}
			}
			row <<= uint(digits*4 - gb.Dx())
			fmt.Fprintf(b, "%0*X\n", digits, row)
		}
		b.WriteString("ENDCHAR\n")
	}
	if notdef != nil {
		char(".notdef", -1, *notdef)
	}
	for _, r := range runes {
		name := fmt.Sprintf("uni%04X", r)
		if r > 0xFFFF {
			name = fmt.Sprintf("u%05X", r)
		}
		char(name, int(r), glyphs[r])
	}
	b.WriteString("ENDFONT\n")
	return b.Flush()
}
