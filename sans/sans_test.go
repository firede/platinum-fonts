package sans_test

import (
	"bytes"
	"flag"
	"image"
	"image/color"
	"image/png"
	"os"
	"slices"
	"testing"

	"github.com/firede/platinum-fonts/bitmap"
	"github.com/firede/platinum-fonts/sans"
	"github.com/firede/platinum-fonts/sanssmall"
)

var update = flag.Bool("update", false, "write the golden images anew")

var styles = []struct {
	name            string
	face            func() *bitmap.Face
	ascent, descent int
}{
	{"sans-regular", sans.Regular, 9, 3},
	{"sans-bold", sans.Bold, 9, 3},
	{"sanssmall-regular", sanssmall.Regular, 8, 3},
	{"sanssmall-bold", sanssmall.Bold, 8, 3},
}

func runes(f *bitmap.Face) []rune {
	var rs []rune
	f.Runes(func(r rune) bool { rs = append(rs, r); return true })
	return rs
}

// TestFamilies checks that the styles of a family draw the same characters,
// with the metrics of their family.
func TestFamilies(t *testing.T) {
	for _, s := range styles {
		f := s.face()
		if f.Ascent() != s.ascent || f.Descent() != s.descent {
			t.Errorf("%s: ascent %d, descent %d", s.name, f.Ascent(), f.Descent())
		}
		for _, r := range "Aaé€�中" {
			if f.Has(r) != (r != '中') {
				t.Errorf("%s draws %q: %v", s.name, r, f.Has(r))
			}
		}
	}
	if !slices.Equal(runes(sans.Regular()), runes(sans.Bold())) {
		t.Error("Platinum Sans Regular and Bold draw different characters")
	}
	if !slices.Equal(runes(sanssmall.Regular()), runes(sanssmall.Bold())) {
		t.Error("Platinum Sans Small Regular and Bold draw different characters")
	}
	// Small Bold is Small Regular emboldened.
	for _, r := range runes(sanssmall.Regular()) {
		reg, _ := sanssmall.Regular().Glyph(r)
		bold, _ := sanssmall.Bold().Glyph(r)
		want := bitmap.EmboldenGlyph(reg)
		if bold.Advance != want.Advance || bold.Bounds != want.Bounds {
			t.Fatalf("Small Bold %q is not Small Regular emboldened", r)
		}
		for y := want.Bounds.Min.Y; y < want.Bounds.Max.Y; y++ {
			for x := want.Bounds.Min.X; x < want.Bounds.Max.X; x++ {
				if bold.At(x, y) != want.At(x, y) {
					t.Fatalf("Small Bold %q is not Small Regular emboldened", r)
				}
			}
		}
	}
}

// TestSamples draws a line of each style and compares it, pixel by pixel,
// with its golden image in testdata.
func TestSamples(t *testing.T) {
	const sample = "Hamburgefonstiv 0123 Ångström Łódź Čeština ⌘↑↵⌫"
	for _, s := range styles {
		f := s.face()
		img := image.NewGray(image.Rect(0, 0, bitmap.Measure(f, sample)+4, 16))
		for i := range img.Pix {
			img.Pix[i] = 0xFF
		}
		bitmap.Draw(img, image.Pt(2, 12), f, sample, color.Black)
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
