package drawn

import (
	"strings"
	"testing"

	"github.com/firede/platinum-fonts/bitmap"
)

// face has a cell of 7 rows, 5 over the baseline, x-height 3.
const face = `@height 7
@ascent 5
@capheight 5
@xheight 3

== space 2

== E
###
#..
###
#..
###

== e
...
...
###
#.#
##.

== i
.#.
...
.#.
.#.
.#.

== a
...
...
##.
.##
###

== c
...
...
###
#..
###

== U+0301
.#
#.

== U+0327
.#
#.

== U+0328 ogonek
.#
.#
`

// picture returns the rows of g from top, over the cell when a mark rises
// there, to the bottom of the cell.
func picture(g bitmap.Glyph) string {
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

func TestParse(t *testing.T) {
	f, err := Parse("test", face)
	if err != nil {
		t.Fatal(err)
	}
	if f.Height != 7 || f.Ascent != 5 || f.XHeight != 3 || f.Tracking != 1 {
		t.Fatalf("metrics %+v", f)
	}
	for _, tc := range []struct {
		r       rune
		advance int
		want    string
	}{
		{' ', 3, ""},
		{'e', 4, ".../.../###/#.#/##./.../.../"},
		// A row clear of a small letter's top, rising over the cell where
		// the cell has no room.
		{'é', 4, ".#./#../.../###/#.#/##./.../.../"},
		// On a capital, rising over the cell.
		{'É', 4, ".#./#../###/#../###/#../###/.../.../"},
		// Under the baseline.
		{'ç', 4, ".../.../###/#../###/.#./#../"},
		// An ogonek under the right side.
		{'ą', 4, ".../.../##./.##/###/..#/..#/"},
		// An i loses its dot under a mark.
		{'í', 4, ".#./#../.../.#./.#./.#./.../.../"},
	} {
		g, ok := f.Glyphs[tc.r]
		if !ok {
			t.Errorf("no %q", tc.r)
			continue
		}
		if g.Advance != tc.advance {
			t.Errorf("%q advances %d, want %d", tc.r, g.Advance, tc.advance)
		}
		if tc.want != "" && picture(g) != tc.want {
			t.Errorf("%q is %s, want %s", tc.r, picture(g), tc.want)
		}
	}
	if _, ok := f.Glyphs[0x301]; ok {
		t.Error("a mark is a glyph")
	}
}

func TestParseErrors(t *testing.T) {
	for name, src := range map[string]string{
		"twice":     "@height 3\n@ascent 2\n== a\n#\n== a\n#\n",
		"outside":   "@height 3\n@ascent 2\n#\n",
		"tall":      "@height 2\n@ascent 1\n== a\n#\n#\n#\n",
		"no height": "@ascent 2\n== a\n#\n",
		"directive": "@height 3\n@ascent 2\n@weight 4\n",
		"name":      "@height 3\n@ascent 2\n== ab\n#\n",
	} {
		if _, err := Parse(name, src); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}
