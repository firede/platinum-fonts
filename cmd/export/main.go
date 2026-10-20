// Command export writes the installable fonts of this repository as BDF:
// each family, with the East Asian glyphs of each region, in each style.
// A family with ligatures has their slices as glyphs of no character, and
// an OpenType feature file next to each font, of the same name, that draws
// them. release/build.py makes the other formats from them.
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
	"github.com/firede/platinum-fonts/internal/ligature"
	"github.com/firede/platinum-fonts/mono"
	"github.com/firede/platinum-fonts/sans"
	"github.com/firede/platinum-fonts/sanssmall"
)

type styles struct{ regular, bold func() *bitmap.Face }

type ligatures struct{ regular, bold func() []mono.Ligature }

type family struct {
	name               string
	latin              styles
	capHeight, xHeight int
	// For a monospaced family, how it sets the East Asian glyphs, and its
	// ligatures.
	eastAsian func(bitmap.Font) bitmap.Font
	ligatures ligatures
}

var families = []family{
	{"Platinum Sans", styles{sans.Regular, sans.Bold}, 9, 6, nil, ligatures{}},
	{"Platinum Sans Small", styles{sanssmall.Regular, sanssmall.Bold}, 8, 5, nil, ligatures{}},
	{"Platinum Mono", styles{mono.Regular, mono.Bold}, 8, 6, mono.EastAsian, ligatures{mono.Ligatures, mono.BoldLigatures}},
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
				ligatures  func() []mono.Ligature
			}{
				{"Regular", fam.latin.regular, reg.cjk.regular, fam.ligatures.regular},
				{"Bold", fam.latin.bold, reg.cjk.bold, fam.ligatures.bold},
			} {
				family := fam.name + " " + reg.suffix
				path := filepath.Join(*out, strings.ReplaceAll(family, " ", "")+"-"+st.name+".bdf")
				var ligs []mono.Ligature
				if st.ligatures != nil {
					ligs = st.ligatures()
				}
				if err := export(path, family, st.name, fam, st.latin(), st.cjk(), ligs); err != nil {
					log.Fatal(err)
				}
				fmt.Println("wrote", path)
			}
		}
	}
}

// export writes the font of family and style: latin's glyphs, and cjk's,
// as fam sets them, where latin has none, and the slices of ligs, with the
// feature that draws them. Its ascent and descent are those of the ink of
// its glyphs.
func export(path, family, style string, fam family, latin, cjk *bitmap.Face, ligs []mono.Ligature) error {
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
	named := map[string]bitmap.Glyph{}
	texts := make([]string, len(ligs))
	for k, l := range ligs {
		texts[k] = l.Text
		for i, g := range l.Glyphs {
			named[sliceName(l.Text, i)] = g
		}
	}
	var b bytes.Buffer
	if err := bdf.Write(&b, h, &notdef, named, glyphs); err != nil {
		return err
	}
	if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
		return err
	}
	if len(ligs) == 0 {
		return nil
	}
	fea := ligature.Feature(texts, bdf.Name, func(k, i int) string { return sliceName(texts[k], i) })
	return os.WriteFile(strings.TrimSuffix(path, ".bdf")+".fea", []byte(fea), 0o644)
}

// sliceName returns the name of the slice of the ligature of text in its
// cell i, as lig.003D_003E.1 for the right of =>.
func sliceName(text string, i int) string {
	var hex []string
	for _, r := range text {
		hex = append(hex, fmt.Sprintf("%04X", r))
	}
	return fmt.Sprintf("lig.%s.%d", strings.Join(hex, "_"), i)
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
