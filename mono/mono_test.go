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
// its character, and advances as far.
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
			if g.Advance != w || !g.Bounds.In(image.Rect(0, -mono.Ascent, w, mono.Descent)) {
				t.Errorf("%q advances %d in %v", r, g.Advance, g.Bounds)
			}
			return true
		})
		for _, r := range "Aa1{}�─█⣿Ａｇ" {
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
		{mono.EastAsian(zhhans.Regular()), '中', 12, image.Rect(0, -9, 11, 2)},
		{mono.EastAsian(zhhans.Bold()), '中', 12, image.Rect(0, -9, 12, 2)},
		{mono.EastAsian(zhhans.Regular()), '，', 12, image.Rect(2, -2, 4, 2)},
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
		img := image.NewGray(image.Rect(0, 0, w+4, 12*len(lines)+4))
		for i := range img.Pix {
			img.Pix[i] = 0xFF
		}
		for i, l := range lines {
			bitmap.Draw(img, image.Pt(2, 2+mono.Ascent+12*i), s.font, l, color.Black)
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
