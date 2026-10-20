package mono

import (
	"image"
	"image/color"
	"image/draw"
	"sync"

	"github.com/firede/platinum-fonts/bitmap"
	"github.com/firede/platinum-fonts/internal/ligature"
)

// Ligature is a ligature of Platinum Mono: Glyphs, a cell each, drawn in
// place of the characters of Text.
type Ligature struct {
	Text   string
	Glyphs []bitmap.Glyph
}

var (
	regularLigatures = sync.OnceValue(func() []Ligature { return ligatures(dataRegularLigatures) })
	boldLigatures    = sync.OnceValue(func() []Ligature { return ligatures(dataBoldLigatures) })
)

// Ligatures returns the ligatures of Regular, in the order Draw tries
// them: longer first.
func Ligatures() []Ligature { return regularLigatures() }

// BoldLigatures returns the ligatures of Bold, in the order Draw tries
// them.
func BoldLigatures() []Ligature { return boldLigatures() }

func ligatures(data []byte) []Ligature {
	f := decode(data)
	ligs := make([]Ligature, len(ligatureTexts))
	n := rune(0)
	for i, t := range ligatureTexts {
		ligs[i].Text = t
		for range []rune(t) {
			g, _ := f.Glyph(n)
			ligs[i].Glyphs = append(ligs[i].Glyphs, g)
			n++
		}
	}
	return ligs
}

// Draw draws s with f as bitmap.Draw does, but the first of ligatures that
// matches in place of its characters, as Platinum Mono's OpenType fonts
// do; none where its first character follows the same character or its
// last is followed by the same, so that runs as ==== stay apart. It
// returns the dot after s, as far as bitmap.Measure tells.
func Draw(dst draw.Image, dot image.Point, f bitmap.Font, ligatures []Ligature, s string, c color.Color) image.Point {
	texts := make([]string, len(ligatures))
	for i, l := range ligatures {
		texts[i] = l.Text
	}
	rs := []rune(s)
	starts := ligature.Starts(texts, rs)
	for i := 0; i < len(rs); {
		if k := starts[i]; k >= 0 {
			for _, g := range ligatures[k].Glyphs {
				dot = bitmap.DrawGlyph(dst, dot, g, c)
			}
			i += len(ligatures[k].Glyphs)
			continue
		}
		if g, ok := f.Glyph(rs[i]); ok {
			dot = bitmap.DrawGlyph(dst, dot, g, c)
		}
		i++
	}
	return dot
}
