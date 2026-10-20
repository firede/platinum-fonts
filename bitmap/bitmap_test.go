package bitmap

import (
	"image"
	"strings"
	"testing"
)

// glyphOf draws rows, '#' ink, with the dot at the left of the row base
// from the top.
func glyphOf(advance, base int, rows ...string) Glyph {
	w := 0
	for _, r := range rows {
		w = max(w, len(r))
	}
	b := image.Rect(0, -base, w, len(rows)-base)
	return NewGlyph(advance, b, func(x, y int) bool {
		r := rows[y+base]
		return x < len(r) && r[x] == '#'
	})
}

func TestRoundTrip(t *testing.T) {
	in := map[rune]Glyph{
		'A': glyphOf(6, 3, ".#.", "#.#", "###", "#.#"),
		'B': glyphOf(6, 3, "##.", "###", "#.#", "##."),
		'D': glyphOf(5, 2, "#", "#", "#"),
		'中': glyphOf(12, 3, "..#..", "#####", "#.#.#", "..#.."),
		' ': {Advance: 4, Bounds: image.Rectangle{}},
	}
	data, err := Encode(10, 2, in)
	if err != nil {
		t.Fatal(err)
	}
	f, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if f.Ascent() != 10 || f.Descent() != 2 || f.Len() != len(in) {
		t.Fatalf("ascent %d, descent %d, %d glyphs", f.Ascent(), f.Descent(), f.Len())
	}
	for r, want := range in {
		got, ok := f.Glyph(r)
		if !ok {
			t.Fatalf("no glyph for %q", r)
		}
		if got.Advance != want.Advance || got.Bounds != want.Bounds {
			t.Errorf("%q: advance %d, bounds %v; want %d, %v", r, got.Advance, got.Bounds, want.Advance, want.Bounds)
		}
		for y := want.Bounds.Min.Y - 1; y <= want.Bounds.Max.Y; y++ {
			for x := want.Bounds.Min.X - 1; x <= want.Bounds.Max.X; x++ {
				if got.At(x, y) != want.At(x, y) {
					t.Fatalf("%q differs at %d, %d", r, x, y)
				}
			}
		}
	}
	for _, r := range "CE\x00" {
		if f.Has(r) {
			t.Errorf("has %q", r)
		}
	}
	var runes []rune
	f.Runes(func(r rune) bool { runes = append(runes, r); return true })
	if string(runes) != " ABD中" {
		t.Errorf("runes %q", string(runes))
	}
}

func TestMask(t *testing.T) {
	g := glyphOf(4, 2, "#.#", ".#.", "#.#")
	m := g.Mask()
	if m.Rect != image.Rect(0, -2, 3, 1) {
		t.Fatalf("mask %v", m.Rect)
	}
	var b strings.Builder
	for y := -2; y < 1; y++ {
		for x := 0; x < 3; x++ {
			if m.AlphaAt(x, y).A == 0xFF {
				b.WriteByte('#')
			} else {
				b.WriteByte('.')
			}
		}
	}
	if b.String() != "#.#.#.#.#" {
		t.Errorf("mask %s", b.String())
	}
}

func TestAdd(t *testing.T) {
	g := glyphOf(4, 2, "#.", ".#")
	m := g.Add(image.Pt(1, 1))
	if m.Advance != 4 || m.Bounds != image.Rect(1, -1, 3, 1) {
		t.Fatalf("advance %d, bounds %v", m.Advance, m.Bounds)
	}
	if !m.At(1, -1) || m.At(2, -1) || m.At(1, 0) || !m.At(2, 0) {
		t.Error("the pixels did not move with the glyph")
	}
}

func TestDecodeRejectsCorrupt(t *testing.T) {
	data, _ := Encode(2, 1, map[rune]Glyph{'x': glyphOf(4, 2, "#.#", ".#.", "#.#")})
	for i := range data {
		if _, err := Decode(data[:i]); err == nil {
			t.Fatalf("decoded a face cut at %d bytes", i)
		}
	}
	if _, err := Decode(append(data, 0)); err == nil {
		t.Fatal("decoded a face with a byte too many")
	}
}
