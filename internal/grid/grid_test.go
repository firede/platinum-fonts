package grid

import (
	"flag"
	"fmt"
	"image"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "write the golden drawing anew")

// TestGlyphs checks that every glyph fills its cell, and compares the
// drawing of all with testdata/grid.txt.
func TestGlyphs(t *testing.T) {
	glyphs := Glyphs()
	if len(glyphs) != 128+32+256+16 {
		t.Errorf("%d glyphs", len(glyphs))
	}
	var b strings.Builder
	for _, r := range slices.Sorted(maps.Keys(glyphs)) {
		g := glyphs[r]
		if g.Advance != W || g.Bounds != image.Rect(0, -Ascent, W, H-Ascent) {
			t.Errorf("%U advances %d in %v", r, g.Advance, g.Bounds)
		}
		fmt.Fprintf(&b, "== U+%04X %c\n", r, r)
		for y := -Ascent; y < H-Ascent; y++ {
			for x := range W {
				if g.At(x, y) {
					b.WriteByte('#')
				} else {
					b.WriteByte('.')
				}
			}
			b.WriteByte('\n')
		}
	}
	const path = "testdata/grid.txt"
	if *update {
		if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (go test -update writes it)", err)
	}
	if string(want) != b.String() {
		t.Errorf("the glyphs differ from %s", path)
	}
}

// TestJoins checks that each line of a box drawing character leaves its
// cell where a line of its weight does, to meet the next cell's.
func TestJoins(t *testing.T) {
	glyphs := Glyphs()
	for r, a := range arms() {
		g := glyphs[r]
		edge := func(n int, at func(i int) bool) []int {
			var in []int
			for i := range n {
				if at(i) {
					in = append(in, i)
				}
			}
			return in
		}
		for _, e := range []struct {
			name      string
			got, want []int
		}{
			{"top", edge(W, func(x int) bool { return g.At(x, -Ascent) }), cols(a[0])},
			{"right", edge(H, func(y int) bool { return g.At(W-1, y-Ascent) }), rows(a[1])},
			{"bottom", edge(W, func(x int) bool { return g.At(x, H-1-Ascent) }), cols(a[2])},
			{"left", edge(H, func(y int) bool { return g.At(0, y-Ascent) }), rows(a[3])},
		} {
			if !slices.Equal(e.got, e.want) {
				t.Errorf("%c (%U) leaves its %s edge at %v, not %v", r, r, e.name, e.got, e.want)
			}
		}
	}
}
