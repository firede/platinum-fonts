package bitmap

import (
	"image"
	"image/color"
	"image/draw"
)

// Font is a source of glyphs: a Face, or fonts combined by Fallback.
type Font interface {
	// Glyph returns the glyph of r, and whether the font has one.
	Glyph(r rune) (Glyph, bool)
}

// Fallback returns a font that draws each character with the first of
// fonts that has it.
func Fallback(fonts ...Font) Font { return fallback(fonts) }

type fallback []Font

func (fb fallback) Glyph(r rune) (Glyph, bool) {
	for _, f := range fb {
		if g, ok := f.Glyph(r); ok {
			return g, true
		}
	}
	return Glyph{}, false
}

// Draw draws s with f in c on dst from dot, skipping the characters f
// lacks, and returns the dot after it.
func Draw(dst draw.Image, dot image.Point, f Font, s string, c color.Color) image.Point {
	for _, r := range s {
		g, ok := f.Glyph(r)
		if !ok {
			continue
		}
		for y := g.Bounds.Min.Y; y < g.Bounds.Max.Y; y++ {
			for x := g.Bounds.Min.X; x < g.Bounds.Max.X; x++ {
				if g.At(x, y) {
					dst.Set(dot.X+x, dot.Y+y, c)
				}
			}
		}
		dot.X += g.Advance
	}
	return dot
}

// Measure returns how far s moves the dot when drawn with f.
func Measure(f Font, s string) int {
	w := 0
	for _, r := range s {
		if g, ok := f.Glyph(r); ok {
			w += g.Advance
		}
	}
	return w
}

// EmboldenGlyph returns g drawn again a pixel to its right: a pixel wider,
// and advancing a pixel more. A glyph without ink is only wider.
func EmboldenGlyph(g Glyph) Glyph {
	b := g.Bounds
	if b.Empty() {
		return Glyph{Advance: g.Advance + 1, Bounds: b}
	}
	b.Max.X++
	return NewGlyph(g.Advance+1, b, func(x, y int) bool { return g.At(x, y) || g.At(x-1, y) })
}

// Embolden returns f with every glyph emboldened by EmboldenGlyph.
func Embolden(f *Face) (*Face, error) {
	glyphs := make(map[rune]Glyph, f.Len())
	f.Runes(func(r rune) bool {
		g, _ := f.Glyph(r)
		glyphs[r] = EmboldenGlyph(g)
		return true
	})
	data, err := Encode(f.Ascent(), f.Descent(), glyphs)
	if err != nil {
		return nil, err
	}
	return Decode(data)
}
