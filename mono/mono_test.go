package mono_test

import (
	"bytes"
	"flag"
	"image"
	"image/color"
	"image/png"
	"os"
	"slices"
	"testing"
	"unicode"

	"github.com/firede/platinum-fonts/bitmap"
	"github.com/firede/platinum-fonts/cjk12/zhhans"
	"github.com/firede/platinum-fonts/internal/cells"
	"github.com/firede/platinum-fonts/mono"
)

var update = flag.Bool("update", false, "write the golden images anew")

const lineHeight = mono.Ascent + mono.Descent

func TestCells(t *testing.T) {
	if mono.UnicodeVersion != cells.UnicodeVersion {
		t.Fatalf("Unicode %s, not %s", mono.UnicodeVersion, cells.UnicodeVersion)
	}
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if mono.Cells(r) != cells.Of(r) {
			t.Fatalf("Cells(%U) = %d, not %d", r, mono.Cells(r), cells.Of(r))
		}
	}
}

func runes(f *bitmap.Face) []rune {
	var rs []rune
	f.Runes(func(r rune) bool { rs = append(rs, r); return true })
	return rs
}

// TestFaces checks that every glyph of both styles stays in the cells of
// its character, and advances as far; one in the Private Use Area, as an
// icon, may draw into the next cell.
func TestFaces(t *testing.T) {
	if !slices.Equal(runes(mono.Regular()), runes(mono.Bold())) {
		t.Error("Regular and Bold draw different characters")
	}
	for _, f := range []*bitmap.Face{mono.Regular(), mono.Bold()} {
		if f.Ascent() != mono.Ascent || f.Descent() != mono.Descent || f.Len() != mono.Glyphs {
			t.Errorf("ascent %d, descent %d, %d glyphs", f.Ascent(), f.Descent(), f.Len())
		}
		f.Runes(func(r rune) bool {
			g, _ := f.Glyph(r)
			w := mono.Cells(r) * mono.CellWidth
			box := image.Rect(0, -mono.Ascent, w, mono.Descent)
			if unicode.Is(unicode.Co, r) {
				box.Max.X += w
			}
			if g.Advance != w || !g.Bounds.In(box) {
				t.Errorf("%q advances %d in %v", r, g.Advance, g.Bounds)
			}
			return true
		})
		for _, r := range "Aa1{}�─█⣿Ａｇ—→■✓❯≠\uE0A0\uF07B\U000F0219" {
			if !f.Has(r) {
				t.Errorf("does not draw %q", r)
			}
		}
		if f.Has('中') || f.Has('，') {
			t.Error("draws East Asian glyphs that cjk12 has")
		}
	}
}

func TestEastAsian(t *testing.T) {
	for _, c := range []struct {
		font    bitmap.Font
		r       rune
		advance int
		bounds  image.Rectangle
	}{
		{mono.EastAsian(zhhans.Regular()), '中', 14, image.Rect(1, -10, 12, 1)},
		{mono.EastAsian(zhhans.Bold()), '中', 14, image.Rect(1, -10, 13, 1)},
		{mono.EastAsian(zhhans.Regular()), '，', 14, image.Rect(3, -3, 5, 1)},
	} {
		g, ok := c.font.Glyph(c.r)
		if !ok || g.Advance != c.advance || g.Bounds != c.bounds {
			t.Errorf("%q advances %d in %v; want %d in %v", c.r, g.Advance, g.Bounds, c.advance, c.bounds)
		}
	}
	// Vertical marks are two cells tall.
	if _, ok := mono.EastAsian(zhhans.Regular()).Glyph('〱'); ok {
		t.Error("〱 does not fit its cells")
	}
	// Both styles draw the same characters.
	regular := bitmap.Fallback(mono.Regular(), mono.EastAsian(zhhans.Regular()))
	bold := bitmap.Fallback(mono.Bold(), mono.EastAsian(zhhans.Bold()))
	zhhans.Regular().Runes(func(r rune) bool {
		_, reg := regular.Glyph(r)
		if _, b := bold.Glyph(r); reg != b {
			t.Errorf("Regular draws %q: %v; Bold: %v", r, reg, b)
		}
		return true
	})
}

// TestSamples draws lines of each style, East Asian glyphs among them,
// and compares them, pixel by pixel, with their golden images in
// testdata.
func TestSamples(t *testing.T) {
	lines := []string{
		"┌─────────┬──────────┐",
		"│ mono.go │ 等宽 Ａｇ│",
		"╞═════════╪══════════╡",
		"│ x := 0; │ ▁▂▃▄▅▆▇█ │",
		"│ (i)=>i+1│ ⣿⡇⢸⠿ ░▒▓ │",
		"└─────────┴──────────┘",
		" main   ",
		"❯ go test ✓ — a→b ≠ ≤ ■●★ ⌘⇧",
	}
	for _, s := range []struct {
		name string
		font bitmap.Font
	}{
		{"regular", bitmap.Fallback(mono.Regular(), mono.EastAsian(zhhans.Regular()))},
		{"bold", bitmap.Fallback(mono.Bold(), mono.EastAsian(zhhans.Bold()))},
	} {
		w := 0
		for _, l := range lines {
			w = max(w, bitmap.Measure(s.font, l))
		}
		img := image.NewGray(image.Rect(0, 0, w+4, lineHeight*len(lines)+4))
		for i := range img.Pix {
			img.Pix[i] = 0xFF
		}
		for i, l := range lines {
			bitmap.Draw(img, image.Pt(2, 2+mono.Ascent+lineHeight*i), s.font, l, color.Black)
		}
		var got bytes.Buffer
		if err := png.Encode(&got, img); err != nil {
			t.Fatal(err)
		}
		path := "testdata/" + s.name + ".png"
		if *update {
			if err := os.WriteFile(path, got.Bytes(), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%v (go test -update writes it)", err)
		}
		if !bytes.Equal(want, got.Bytes()) {
			t.Errorf("%s differs from %s", s.name, path)
		}
	}
}

// TestLigatures checks that each ligature draws a cell for each of its
// characters, and compares lines of them with testdata/ligatures.png.
func TestLigatures(t *testing.T) {
	cell := image.Rect(0, -mono.Ascent, mono.CellWidth, mono.Descent)
	for _, ligs := range [][]mono.Ligature{mono.Ligatures(), mono.BoldLigatures()} {
		if len(ligs) != 31 {
			t.Errorf("%d ligatures", len(ligs))
		}
		for i, l := range ligs {
			if i > 0 && len([]rune(l.Text)) > len([]rune(ligs[i-1].Text)) {
				t.Errorf("%q is tried after the shorter %q", l.Text, ligs[i-1].Text)
			}
			if len(l.Glyphs) != len([]rune(l.Text)) {
				t.Errorf("%q has %d glyphs", l.Text, len(l.Glyphs))
			}
			for _, g := range l.Glyphs {
				if g.Advance != mono.CellWidth || !g.Bounds.In(cell) {
					t.Errorf("a slice of %q advances %d in %v", l.Text, g.Advance, g.Bounds)
				}
			}
		}
	}

	lines := []string{
		"a == b != c <= d >= e => f -> g <- h",
		"x === y !== z <=> <-> --> <-- ==> <==",
		"... /// /** **/ ::= <> |> <| ++ -- // /* */ := ::",
		"<!-- a --> 不连：==== .... ---> a:::b",
	}
	for _, s := range []struct {
		font bitmap.Font
		ligs []mono.Ligature
	}{
		{mono.Regular(), mono.Ligatures()},
		{mono.Bold(), mono.BoldLigatures()},
	} {
		for _, l := range lines {
			f := bitmap.Fallback(s.font, mono.EastAsian(zhhans.Regular()))
			img := image.NewGray(image.Rect(0, 0, 1, 1))
			if got, want := mono.Draw(img, image.Point{}, f, s.ligs, l, color.Black).X, bitmap.Measure(f, l); got != want {
				t.Errorf("%q: Draw goes %d, Measure %d", l, got, want)
			}
		}
	}

	w := 0
	for _, l := range lines {
		w = max(w, bitmap.Measure(mono.Regular(), l)+2*mono.CellWidth*4)
	}
	img := image.NewGray(image.Rect(0, 0, w+4, 2*lineHeight*len(lines)+4))
	for i := range img.Pix {
		img.Pix[i] = 0xFF
	}
	for i, l := range lines {
		y := 2 + mono.Ascent + 2*lineHeight*i
		mono.Draw(img, image.Pt(2, y), bitmap.Fallback(mono.Regular(), mono.EastAsian(zhhans.Regular())), mono.Ligatures(), l, color.Black)
		mono.Draw(img, image.Pt(2, y+lineHeight), bitmap.Fallback(mono.Bold(), mono.EastAsian(zhhans.Bold())), mono.BoldLigatures(), l, color.Black)
	}
	var got bytes.Buffer
	if err := png.Encode(&got, img); err != nil {
		t.Fatal(err)
	}
	const path = "testdata/ligatures.png"
	if *update {
		if err := os.WriteFile(path, got.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (go test -update writes it)", err)
	}
	if !bytes.Equal(want, got.Bytes()) {
		t.Errorf("the ligatures differ from %s", path)
	}
}
