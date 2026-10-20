// Command export writes the installable fonts of this repository as BDF:
// each family, with the East Asian glyphs of each region, in each style.
// release/build.py makes the other formats from them.
//
//	go run ./cmd/export -o dist/bdf
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/firede/platinum-fonts/bitmap"
	"github.com/firede/platinum-fonts/cjk12/ja"
	"github.com/firede/platinum-fonts/cjk12/ko"
	"github.com/firede/platinum-fonts/cjk12/zhhans"
	"github.com/firede/platinum-fonts/cjk12/zhhant"
	"github.com/firede/platinum-fonts/internal/bdf"
	"github.com/firede/platinum-fonts/mono"
	"github.com/firede/platinum-fonts/sans"
	"github.com/firede/platinum-fonts/sanssmall"
)

type styles struct{ regular, bold func() *bitmap.Face }

type family struct {
	name               string
	latin              styles
	capHeight, xHeight int
	// For a monospaced family, how it sets the East Asian glyphs.
	eastAsian func(bitmap.Font) bitmap.Font
}

var families = []family{
	{"Platinum Sans", styles{sans.Regular, sans.Bold}, 9, 6, nil},
	{"Platinum Sans Small", styles{sanssmall.Regular, sanssmall.Bold}, 8, 5, nil},
	{"Platinum Mono", styles{mono.Regular, mono.Bold}, 8, 6, mono.EastAsian},
}

var regions = []struct {
	suffix string
	cjk    styles
}{
	{"SC", styles{zhhans.Regular, zhhans.Bold}},
	{"TC", styles{zhhant.Regular, zhhant.Bold}},
	{"JP", styles{ja.Regular, ja.Bold}},
	{"KR", styles{ko.Regular, ko.Bold}},
}

// pixelSize is the em of every font, in pixels: the East Asian grid.
const pixelSize = 12

func main() {
	out := flag.String("o", "dist/bdf", "the directory to write the fonts in")
	flag.Parse()
	log.SetFlags(0)
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	for _, fam := range families {
		for _, reg := range regions {
			for _, st := range []struct {
				name       string
				latin, cjk func() *bitmap.Face
			}{
				{"Regular", fam.latin.regular, reg.cjk.regular},
				{"Bold", fam.latin.bold, reg.cjk.bold},
			} {
				family := fam.name + " " + reg.suffix
				path := filepath.Join(*out, strings.ReplaceAll(family, " ", "")+"-"+st.name+".bdf")
				if err := export(path, family, st.name, fam, st.latin(), st.cjk()); err != nil {
					log.Fatal(err)
				}
				fmt.Println("wrote", path)
			}
		}
	}
}

// export writes the font of family and style: latin's glyphs, and cjk's,
// as fam sets them, where latin has none. Its ascent and descent are those
// of the ink of its glyphs.
func export(path, family, style string, fam family, latin, cjk *bitmap.Face) error {
	glyphs := map[rune]bitmap.Glyph{}
	ascent, descent := 0, 0
	add := func(f *bitmap.Face, font bitmap.Font) {
		f.Runes(func(r rune) bool {
			if _, ok := glyphs[r]; ok {
				return true
			}
			g, ok := font.Glyph(r)
			if !ok {
				return true
			}
			glyphs[r] = g
			if r != 0x3031 && r != 0x3032 { // marks of vertical text, two cells tall
				top, bottom := ink(g)
				ascent, descent = max(ascent, top), max(descent, bottom)
			}
			return true
		})
	}
	spacing := "P"
	var cjkFont bitmap.Font = cjk
	if fam.eastAsian != nil {
		spacing, cjkFont = "M", fam.eastAsian(cjk)
	}
	add(latin, latin)
	add(cjk, cjkFont)
	notdef, ok := latin.Glyph(0xFFFD)
	if !ok {
		return fmt.Errorf("%s %s has no U+FFFD for .notdef", family, style)
	}
	h := bdf.Header{
		Name:      fmt.Sprintf("-Platinum-%s-%s-R-Normal--%d-%d-75-75-%s-0-ISO10646-1", family, style, pixelSize, pixelSize*10, spacing),
		PixelSize: pixelSize,
		Ascent:    ascent,
		Descent:   descent,
		Properties: map[string]string{
			"FAMILY_NAME":  fmt.Sprintf("%q", family),
			"WEIGHT_NAME":  fmt.Sprintf("%q", style),
			"SPACING":      fmt.Sprintf("%q", spacing),
			"CAP_HEIGHT":   fmt.Sprint(fam.capHeight),
			"X_HEIGHT":     fmt.Sprint(fam.xHeight),
			"DEFAULT_CHAR": fmt.Sprint(0xFFFD),
		},
	}
	var b bytes.Buffer
	if err := bdf.Write(&b, h, &notdef, glyphs); err != nil {
		return err
	}
	return os.WriteFile(path, b.Bytes(), 0o644)
}

// ink returns how many rows g's ink rises over the baseline and goes under
// it.
func ink(g bitmap.Glyph) (top, bottom int) {
	for y := g.Bounds.Min.Y; y < g.Bounds.Max.Y; y++ {
		for x := g.Bounds.Min.X; x < g.Bounds.Max.X; x++ {
			if g.At(x, y) {
				top, bottom = max(top, -y), max(bottom, y+1)
			}
		}
	}
	return top, bottom
}
