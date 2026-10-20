package main

import (
	"fmt"
	"image"
	"os"
	"text/template"
	"unicode"

	"github.com/firede/platinum-fonts/bitmap"
	"github.com/firede/platinum-fonts/internal/bdf"
	"github.com/firede/platinum-fonts/internal/cells"
	"github.com/firede/platinum-fonts/internal/drawn"
	"github.com/firede/platinum-fonts/internal/grid"
	"github.com/firede/platinum-fonts/internal/source"
)

// monoDrawn is the drawing of the glyphs Platinum Mono does not take from
// its source.
const monoDrawn = "glyphs/mono.txt"

// monoMade is what buildMono made, for the docs.
type monoMade struct {
	src           *source.Source
	regular, bold *bitmap.Face
}

// buildMono makes Platinum Mono. Its glyphs, the first that has one:
// those drawn in glyphs/mono.txt; the box drawing, blocks, braille and
// Powerline separators of package grid; and the glyphs of Fusion Pixel
// Font's 12px monospaced fonts, set in the cells of their characters. Of
// the East Asian blocks it keeps only the glyphs that differ from cjk12's
// set a row lower, as mono.EastAsian sets them, and those that would not
// fit their cells bold; they must be the same in every region.
func buildMono(sources map[string]*source.Source, cache string, cjk *cjkMade, out map[string][]byte) (*monoMade, error) {
	src := sources["fusion-pixel-12-monospaced"]
	if src == nil {
		return nil, fmt.Errorf("sources.toml has no fusion-pixel-12-monospaced")
	}
	data, err := source.Fetch(src, cache)
	if err != nil {
		return nil, err
	}
	archive, err := source.OpenArchive(data)
	if err != nil {
		return nil, err
	}
	parse := func(flavour string) (*bdf.Font, error) {
		raw, err := archive.File("fusion-pixel-12px-monospaced-" + flavour + ".bdf")
		if err != nil {
			return nil, err
		}
		font, err := bdf.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", flavour, err)
		}
		if font.Ascent != grid.Ascent || font.Descent != grid.H-grid.Ascent {
			return nil, fmt.Errorf("%s: ascent %d and descent %d, not the grid's", flavour, font.Ascent, font.Descent)
		}
		return font, nil
	}

	glyphs := map[rune]bitmap.Glyph{}
	latin, err := parse("latin")
	if err != nil {
		return nil, err
	}
	for r, g := range latin.Glyphs {
		if inCJK(r) || unicode.Is(unicode.Co, r) || noncharacter(r) {
			continue
		}
		if g, ok := fit(r, g); ok {
			glyphs[r] = g
		}
	}

	// The East Asian glyphs Mono sets otherwise than cjk12.
	differ := map[rune]bitmap.Glyph{}
	for i, fl := range cjkFlavours {
		font, err := parse(fl.Flavour)
		if err != nil {
			return nil, err
		}
		face := cjk.faces[i].face
		region := map[rune]bitmap.Glyph{}
		for r, g := range font.Glyphs {
			if !inCJK(r) {
				continue
			}
			// A glyph as wide as its cells would grow out of them in
			// cjk12's Bold; Mono emboldens it within them.
			c, ok := face.Glyph(r)
			if !ok || !sameInk(g, c.Add(image.Pt(0, 1))) || c.Bounds.Max.X >= cells.Of(r)*grid.W {
				region[r] = g
			}
		}
		if i == 0 {
			differ = region
			continue
		}
		if len(region) != len(differ) {
			return nil, fmt.Errorf("%s: %d East Asian glyphs differ from cjk12's, %d in %s", fl.Flavour, len(region), len(differ), cjkFlavours[0].Flavour)
		}
		for r, g := range region {
			if d, ok := differ[r]; !ok || !sameInk(g, d) {
				return nil, fmt.Errorf("%s: %U differs from cjk12 otherwise than in %s", fl.Flavour, r, cjkFlavours[0].Flavour)
			}
		}
	}
	for r, g := range differ {
		if g, ok := fit(r, g); ok {
			glyphs[r] = g
		}
	}

	generated := grid.Glyphs()
	for r, g := range generated {
		glyphs[r] = g
	}

	text, err := os.ReadFile(monoDrawn)
	if err != nil {
		return nil, err
	}
	own, err := drawn.Parse(monoDrawn, string(text))
	if err != nil {
		return nil, err
	}
	if own.Height != grid.H || own.Ascent != grid.Ascent || own.Tracking != 0 {
		return nil, fmt.Errorf("%s: not drawn on the grid: height %d, ascent %d, tracking %d", monoDrawn, own.Height, own.Ascent, own.Tracking)
	}
	for r, g := range own.Glyphs {
		if g.Advance != cells.Of(r)*grid.W {
			return nil, fmt.Errorf("%s: %q is %d pixels wide; it takes %d cells", monoDrawn, r, g.Advance, cells.Of(r))
		}
		glyphs[r] = g
	}

	bold := map[rune]bitmap.Glyph{}
	for r, g := range glyphs {
		_, gen := generated[r]
		if _, mine := own.Glyphs[r]; gen && !mine {
			bold[r] = g
		} else {
			bold[r] = emboldenInCells(g)
		}
	}
	made := &monoMade{src: src}
	for _, st := range []struct {
		file   string
		glyphs map[rune]bitmap.Glyph
		face   **bitmap.Face
	}{
		{"mono/regular.bin", glyphs, &made.regular},
		{"mono/bold.bin", bold, &made.bold},
	} {
		packed, err := bitmap.Encode(grid.Ascent, grid.H-grid.Ascent, st.glyphs)
		if err != nil {
			return nil, err
		}
		if *st.face, err = bitmap.Decode(packed); err != nil {
			return nil, err
		}
		out[st.file] = packed
	}

	code, err := generate(monoTemplate, map[string]any{
		"Version": src.Pinned(), "Glyphs": len(glyphs), "Width": grid.W,
		"CapHeight": own.CapHeight, "XHeight": own.XHeight,
	})
	if err != nil {
		return nil, err
	}
	out["mono/mono.go"] = code
	if out["mono/table.go"], err = generate(cellsTemplate, map[string]any{
		"Unicode": cells.UnicodeVersion, "Ranges": cells.Ranges(),
	}); err != nil {
		return nil, err
	}
	return made, nil
}

// fit returns g set in the cells r takes: as it is if it fits them, its
// ink moved to their middle if it fits only so, or false if it does not
// fit. The last column of the cells is left for the space between
// characters when the ink is moved.
func fit(r rune, g bitmap.Glyph) (bitmap.Glyph, bool) {
	w := cells.Of(r) * grid.W
	ink := inkBox(g)
	if ink.Empty() {
		return bitmap.Glyph{Advance: w}, true
	}
	if ink.Min.Y < -grid.Ascent || ink.Max.Y > grid.H-grid.Ascent {
		return bitmap.Glyph{}, false
	}
	if g.Advance == w && ink.Min.X >= 0 && ink.Max.X <= w {
		return g, true
	}
	if ink.Dx() > w-1 {
		return bitmap.Glyph{}, false
	}
	g = g.Add(image.Pt((w-1-ink.Dx())/2-ink.Min.X, 0))
	g.Advance = w
	return g, true
}

// emboldenInCells returns g emboldened by bitmap.EmboldenGlyphOpen,
// advancing as far as g: what would leave its cells is left out.
func emboldenInCells(g bitmap.Glyph) bitmap.Glyph {
	b := bitmap.EmboldenGlyphOpen(g)
	box := b.Bounds.Intersect(image.Rect(0, b.Bounds.Min.Y, g.Advance, b.Bounds.Max.Y))
	return bitmap.NewGlyph(g.Advance, box, b.At)
}

// inkBox returns the box of g's ink.
func inkBox(g bitmap.Glyph) image.Rectangle {
	var box image.Rectangle
	for y := g.Bounds.Min.Y; y < g.Bounds.Max.Y; y++ {
		for x := g.Bounds.Min.X; x < g.Bounds.Max.X; x++ {
			if g.At(x, y) {
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return box
}

// sameInk reports whether a and b ink the same pixels.
func sameInk(a, b bitmap.Glyph) bool {
	box := inkBox(a)
	if box != inkBox(b) {
		return false
	}
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			if a.At(x, y) != b.At(x, y) {
				return false
			}
		}
	}
	return true
}

func noncharacter(r rune) bool { return r&0xFFFE == 0xFFFE || r >= 0xFDD0 && r <= 0xFDEF }

var monoTemplate = template.Must(template.New("mono").Parse(`// Code generated by go run ./cmd/build; DO NOT EDIT.

// Package mono contains Platinum Mono, a face for terminals and code on a
// grid of cells {{.Width}}px wide and 12px tall: ascent 10, cap height {{.CapHeight}}, x-height {{.XHeight}}.
// A character takes the cells Cells reports, and its glyph advances as far.
// Styles: Regular, Bold.
//
// Its Latin and symbols are Fusion Pixel Font {{.Version}}'s (monospaced);
// its box drawing, blocks, braille and Powerline separators are drawn to
// fill their cells and meet the cells around them. Its East Asian glyphs
// are those of cjk12, set in its cells by EastAsian.
//
// Bold is Regular emboldened within the cells: each glyph drawn again a
// pixel to its right, gaps of a pixel between strokes kept open. The
// characters that fill their cells are the same in both styles.
// Glyph data: SIL Open Font License 1.1; see NOTICE.md.
package mono

import (
	_ "embed"
	"sync"

	"github.com/firede/platinum-fonts/bitmap"
)

// Glyphs is the number of characters each face draws.
const Glyphs = {{.Glyphs}}

//go:embed regular.bin
var dataRegular []byte

//go:embed bold.bin
var dataBold []byte

var (
	faceRegular = sync.OnceValue(func() *bitmap.Face { return decode(dataRegular) })
	faceBold    = sync.OnceValue(func() *bitmap.Face { return decode(dataBold) })
)

// Regular returns Platinum Mono Regular.
func Regular() *bitmap.Face { return faceRegular() }

// Bold returns Platinum Mono Bold.
func Bold() *bitmap.Face { return faceBold() }

func decode(data []byte) *bitmap.Face {
	f, err := bitmap.Decode(data)
	if err != nil {
		panic(err)
	}
	return f
}
`))

var cellsTemplate = template.Must(template.New("cells").Parse(`// Code generated by go run ./cmd/build; DO NOT EDIT.

package mono

// UnicodeVersion is the version of Unicode Cells follows.
const UnicodeVersion = {{printf "%q" .Unicode}}

// cellRanges are the runs of characters that take other than one cell, in
// order.
var cellRanges = []struct {
	lo, hi rune
	cells  uint8
}{
{{- range .Ranges}}
	{ {{- printf "0x%04X" .Lo}}, {{printf "0x%04X" .Hi}}, {{.Cells -}} },
{{- end}}
}
`))
