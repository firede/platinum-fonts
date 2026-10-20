package mono

import (
	"image"
	"sort"

	"github.com/firede/platinum-fonts/bitmap"
)

// The grid: cells CellWidth pixels wide, Ascent rows over the baseline
// and Descent under it.
const (
	CellWidth = 6
	Ascent    = 10
	Descent   = 2
)

// Cells returns the cells r takes, as wcwidth computes it: 2 for East
// Asian Wide and Fullwidth characters; 0 for marks, format characters,
// controls and the medial vowels and final consonants of Hangul; 1 for
// every other, East Asian Ambiguous among them.
func Cells(r rune) int {
	i := sort.Search(len(cellRanges), func(i int) bool { return cellRanges[i].hi >= r })
	if i < len(cellRanges) && cellRanges[i].lo <= r {
		return int(cellRanges[i].cells)
	}
	return 1
}

// EastAsian returns the glyphs of f, a face of cjk12, as Mono sets them: a
// row lower, each advancing the cells of its character. It leaves out the
// glyphs that do not fit their cells. Regular and Bold draw those that
// Mono sets otherwise, so they go first:
//
//	text := bitmap.Fallback(mono.Regular(), mono.EastAsian(zhhans.Regular()))
//	bold := bitmap.Fallback(mono.Bold(), mono.EastAsian(zhhans.Bold()))
func EastAsian(f bitmap.Font) bitmap.Font { return eastAsian{f} }

type eastAsian struct{ f bitmap.Font }

func (e eastAsian) Glyph(r rune) (bitmap.Glyph, bool) {
	g, ok := e.f.Glyph(r)
	if !ok {
		return bitmap.Glyph{}, false
	}
	g = g.Add(image.Pt(0, 1))
	g.Advance = Cells(r) * CellWidth
	if !g.Bounds.In(image.Rect(0, -Ascent, g.Advance, Descent)) {
		return bitmap.Glyph{}, false
	}
	return g, true
}
