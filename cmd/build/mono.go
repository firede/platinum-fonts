package main

import (
	"fmt"
	"image"
	"os"
	"slices"
	"text/template"
	"unicode"

	"github.com/firede/platinum-fonts/bitmap"
	"github.com/firede/platinum-fonts/internal/bdf"
	"github.com/firede/platinum-fonts/internal/cells"
	"github.com/firede/platinum-fonts/internal/drawn"
	"github.com/firede/platinum-fonts/internal/grid"
	"github.com/firede/platinum-fonts/internal/ligature"
	"github.com/firede/platinum-fonts/internal/source"
)

// monoDrawn is the drawing of the glyphs Platinum Mono does not take from
// its source.
const monoDrawn = "glyphs/mono.txt"

// monoLigatureLicense is where buildMono writes the license of the
// ligatures.
const monoLigatureLicense = "licenses/akr-pixel/LICENSE.txt"

// The grid Fusion Pixel's monospaced fonts and AKR Pixel's ligatures are
// drawn on: cells srcW pixels wide and srcH tall, srcAscent of them over
// the baseline. Mono's cells, of package grid, are a column wider and a
// row taller either side; widen sets glyphs in them.
const srcW, srcH, srcAscent = 6, 12, 10

// monoMade is what buildMono made, for the docs.
type monoMade struct {
	src, ligSrc   *source.Source
	regular, bold *bitmap.Face
	// The ligatures, longest first, and their slices in each style.
	ligTexts            []string
	ligRegular, ligBold [][]bitmap.Glyph
}

// buildMono makes Platinum Mono. Its glyphs, the first that has one:
// those drawn in glyphs/mono.txt; the box drawing, blocks, braille and
// Powerline separators of package grid; and the glyphs of Fusion Pixel
// Font's 12px monospaced fonts, set in the cells of their characters and
// widened. Of the East Asian blocks it keeps only the glyphs that differ
// from cjk12's set a row lower, as Fusion Pixel sets them: its fullwidth
// letters and digits, on the baseline. They must be the same in every
// region.
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
		if font.Ascent != srcAscent || font.Descent != srcH-srcAscent {
			return nil, fmt.Errorf("%s: ascent %d and descent %d, not %d and %d", flavour, font.Ascent, font.Descent, srcAscent, srcH-srcAscent)
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
			glyphs[r] = widen(g)
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
			c, ok := face.Glyph(r)
			if !ok || !sameInk(g, c.Add(image.Pt(0, 1))) {
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
			glyphs[r] = widen(g)
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

	made := &monoMade{src: src}
	if err := buildLigatures(made, sources, cache, glyphs, out); err != nil {
		return nil, err
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
		"Version": src.Pinned(), "Glyphs": len(glyphs),
		"Width": grid.W, "Height": grid.H, "Ascent": grid.Ascent,
		"Ligatures": made.ligTexts, "LigatureSource": made.ligSrc.Repository,
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

// buildLigatures makes the ligatures of AKR Pixel, sliced into cells, for
// the faces of glyphs: Bold's emboldened whole, then sliced.
func buildLigatures(made *monoMade, sources map[string]*source.Source, cache string, glyphs map[rune]bitmap.Glyph, out map[string][]byte) error {
	fetch := func(name string) (*source.Source, []byte, error) {
		src := sources[name]
		if src == nil {
			return nil, nil, fmt.Errorf("sources.toml has no %s", name)
		}
		data, err := source.Fetch(src, cache)
		return src, data, err
	}
	src, text, err := fetch("akr-pixel-ligatures")
	if err != nil {
		return err
	}
	_, license, err := fetch("akr-pixel-license")
	if err != nil {
		return err
	}
	out[monoLigatureLicense] = license
	ligs, err := ligature.Parse(src.Name, string(text), srcW, srcH, srcAscent)
	if err != nil {
		return err
	}
	for _, l := range ligs {
		for _, r := range l.Text {
			if _, ok := glyphs[r]; !ok || cells.Of(r) != 1 {
				return fmt.Errorf("%s: %q has %q, which Mono does not draw in a cell", src.Name, l.Text, r)
			}
		}
	}
	slices.SortStableFunc(ligs, func(a, b ligature.Ligature) int {
		return len([]rune(b.Text)) - len([]rune(a.Text))
	})

	made.ligSrc = src
	regular, bold := map[rune]bitmap.Glyph{}, map[rune]bitmap.Glyph{}
	n := rune(0)
	for _, l := range ligs {
		made.ligTexts = append(made.ligTexts, l.Text)
		g := widenCells(l.Glyph)
		b := emboldenInCells(g)
		var rs, bs []bitmap.Glyph
		for i := range len([]rune(l.Text)) {
			rs = append(rs, ligature.Slice(g, i, grid.W))
			bs = append(bs, ligature.Slice(b, i, grid.W))
			regular[n], bold[n] = rs[i], bs[i]
			n++
		}
		made.ligRegular = append(made.ligRegular, rs)
		made.ligBold = append(made.ligBold, bs)
	}
	for file, style := range map[string]map[rune]bitmap.Glyph{"mono/regular-ligatures.bin": regular, "mono/bold-ligatures.bin": bold} {
		if out[file], err = bitmap.Encode(grid.Ascent, grid.H-grid.Ascent, style); err != nil {
			return err
		}
	}
	return nil
}

// fit returns g, drawn on cells of srcW, set in the cells r takes: as it
// is if it fits them, its ink moved to their middle if it fits only so, or
// false if it does not fit. The last column of the cells is left for the
// space between characters when the ink is moved.
func fit(r rune, g bitmap.Glyph) (bitmap.Glyph, bool) {
	w := cells.Of(r) * srcW
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

// widen returns g, set in cells of srcW, in cells of grid.W: its ink a
// column to the right, so that the first column of each character stays
// clear between it and the one before; the columns the cells gain, on the
// left and the right, inked where g's ink runs from its last column into
// its first, as where it meets a copy of itself.
func widen(g bitmap.Glyph) bitmap.Glyph {
	w := g.Advance / srcW * grid.W
	joins := func(y int) bool { return g.At(0, y) && g.At(g.Advance-1, y) }
	box := image.Rect(0, g.Bounds.Min.Y, w, g.Bounds.Max.Y)
	return bitmap.NewGlyph(w, box, func(x, y int) bool {
		if x == 0 || x > g.Advance {
			return joins(y)
		}
		return g.At(x-1, y)
	})
}

// widenCells returns g, a ligature drawn on cells of srcW, on cells of
// grid.W: each cell's ink a column to the right, and the column it gains
// on its left inked where the ink runs on from the cell before.
func widenCells(g bitmap.Glyph) bitmap.Glyph {
	n := g.Advance / srcW
	box := image.Rect(0, g.Bounds.Min.Y, n*grid.W, g.Bounds.Max.Y)
	return bitmap.NewGlyph(n*grid.W, box, func(x, y int) bool {
		i, c := x/grid.W, x%grid.W
		if c == 0 {
			return i > 0 && g.At(i*srcW-1, y) && g.At(i*srcW, y)
		}
		return g.At(i*srcW+c-1, y)
	})
}

// emboldenInCells returns g emboldened by bitmap.EmboldenGlyphOpen,
// advancing as far as g: what would leave its cells is left out. The
// first column of the cells stays clear, as in g, to keep characters
// apart.
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
// grid of cells {{.Width}}px wide and {{.Height}}px tall: ascent {{.Ascent}}, cap height {{.CapHeight}}, x-height {{.XHeight}}.
// A character takes the cells Cells reports, and its glyph advances as far.
// Styles: Regular, Bold.
//
// Its Latin and symbols are Fusion Pixel Font {{.Version}}'s (monospaced),
// a column wider; its box drawing, blocks, braille and Powerline
// separators are drawn to fill their cells and meet the cells around them.
// Its East Asian glyphs are those of cjk12, set in its cells by EastAsian.
//
// Draw draws its ligatures, hand-drawn for AKR Pixel ({{.LigatureSource}}),
// widened likewise: each a slice in each cell of its characters.
//
// Bold is Regular emboldened within the cells: each glyph drawn again a
// pixel to its right, gaps of a pixel between strokes kept open, the
// first column of the cells clear. The characters that fill their cells
// are the same in both styles.
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

//go:embed regular-ligatures.bin
var dataRegularLigatures []byte

//go:embed bold-ligatures.bin
var dataBoldLigatures []byte

// ligatureTexts are the characters of the ligatures, in the order Draw
// tries them; the slices of each follow those of the one before.
var ligatureTexts = []string{
{{- range .Ligatures}}
	{{printf "%q" .}},
{{- end}}
}

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
