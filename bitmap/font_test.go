package bitmap

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func face(t *testing.T, glyphs map[rune]Glyph) *Face {
	t.Helper()
	data, err := Encode(9, 3, glyphs)
	if err != nil {
		t.Fatal(err)
	}
	f, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// picture returns the rows of g's bounds, '#' ink.
func picture(g Glyph) string {
	var b strings.Builder
	for y := g.Bounds.Min.Y; y < g.Bounds.Max.Y; y++ {
		for x := g.Bounds.Min.X; x < g.Bounds.Max.X; x++ {
			if g.At(x, y) {
				b.WriteByte('#')
			} else {
				b.WriteByte('.')
			}
		}
		b.WriteByte('/')
	}
	return b.String()
}

func TestEmbolden(t *testing.T) {
	g := EmboldenGlyph(glyphOf(4, 2, "#.#", ".#.", "#.#"))
	if g.Advance != 5 || g.Bounds != image.Rect(0, -2, 4, 1) {
		t.Fatalf("advance %d, bounds %v", g.Advance, g.Bounds)
	}
	if p := picture(g); p != "####/.##./####/" {
		t.Errorf("emboldened %s", p)
	}
	space := EmboldenGlyph(Glyph{Advance: 4})
	if space.Advance != 5 || !space.Bounds.Empty() {
		t.Errorf("an emboldened space advances %d in %v", space.Advance, space.Bounds)
	}
	f := face(t, map[rune]Glyph{'x': glyphOf(4, 2, "#.#", ".#.", "#.#"), ' ': {Advance: 3}})
	bold, err := Embolden(f)
	if err != nil {
		t.Fatal(err)
	}
	if bold.Len() != 2 || bold.Ascent() != 9 || bold.Descent() != 3 {
		t.Fatalf("%d glyphs, ascent %d, descent %d", bold.Len(), bold.Ascent(), bold.Descent())
	}
	if g, _ := bold.Glyph('x'); picture(g) != "####/.##./####/" {
		t.Errorf("bold x %s", picture(g))
	}
}

func TestFallbackDrawMeasure(t *testing.T) {
	latin := face(t, map[rune]Glyph{'a': glyphOf(3, 1, "##", "##")})
	cjk := face(t, map[rune]Glyph{'a': glyphOf(9, 1, "#"), '中': glyphOf(5, 1, "####", "#..#")})
	f := Fallback(latin, cjk)
	if g, _ := f.Glyph('a'); g.Advance != 3 {
		t.Errorf("a comes from the second font")
	}
	if g, ok := f.Glyph('中'); !ok || g.Advance != 5 {
		t.Errorf("中 does not fall back")
	}
	if _, ok := f.Glyph('b'); ok {
		t.Error("a glyph for b")
	}
	if w := Measure(f, "a中b"); w != 8 {
		t.Errorf("measures %d", w)
	}
	img := image.NewGray(image.Rect(0, 0, 10, 3))
	end := Draw(img, image.Pt(1, 2), f, "a中", color.White)
	if end != image.Pt(9, 2) {
		t.Errorf("ends at %v", end)
	}
	var b strings.Builder
	for y := 0; y < 3; y++ {
		for x := 0; x < 10; x++ {
			if img.GrayAt(x, y).Y != 0 {
				b.WriteByte('#')
			} else {
				b.WriteByte('.')
			}
		}
		b.WriteByte('/')
	}
	if b.String() != "........../.##.####../.##.#..#../" {
		t.Errorf("drew %s", b.String())
	}
}
