package cjk12_test

import (
	"bytes"
	"flag"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/firede/platinum-fonts/bitmap"
	"github.com/firede/platinum-fonts/cjk12/ja"
	"github.com/firede/platinum-fonts/cjk12/ko"
	"github.com/firede/platinum-fonts/cjk12/zhhans"
	"github.com/firede/platinum-fonts/cjk12/zhhant"
)

var update = flag.Bool("update", false, "write the golden images anew")

var faces = []struct {
	name   string
	face   func() *bitmap.Face
	bold   func() *bitmap.Face
	glyphs int
	sample string
}{
	{"zhhans", zhhans.Regular, zhhans.Bold, zhhans.Glyphs, "永和九年，岁在癸丑。藏鑫覆盖"},
	{"zhhant", zhhant.Regular, zhhant.Bold, zhhant.Glyphs, "永和九年，歲在癸丑。藏鑫覆蓋"},
	{"ja", ja.Regular, ja.Bold, ja.Glyphs, "いろはにほへと　カタカナ　鬱"},
	{"ko", ko.Regular, ko.Bold, ko.Glyphs, "다람쥐　헌　쳇바퀴에　타고파"},
}

func TestFaces(t *testing.T) {
	for _, f := range faces {
		face := f.face()
		if face.Len() != f.glyphs {
			t.Errorf("%s: %d glyphs, the package says %d", f.name, face.Len(), f.glyphs)
		}
		// Ideographs ink 11×11 from a row under the baseline, and advance 12.
		g, ok := face.Glyph('中')
		if !ok || g.Advance != 12 || g.Bounds != image.Rect(0, -10, 11, 1) {
			t.Errorf("%s: 中 advances %d in %v", f.name, g.Advance, g.Bounds)
		}
		// Bold draws the same characters, each a pixel wider.
		b, ok := f.bold().Glyph('中')
		if !ok || b.Advance != 13 || b.Bounds != image.Rect(0, -10, 12, 1) || f.bold().Len() != f.glyphs {
			t.Errorf("%s: bold 中 advances %d in %v, of %d glyphs", f.name, b.Advance, b.Bounds, f.bold().Len())
		}
		for _, r := range "Aa1 ," {
			if face.Has(r) {
				t.Errorf("%s draws %q, which Latin faces draw", f.name, r)
			}
		}
	}
}

// TestSamples draws a line of each face and compares it, pixel by pixel,
// with its golden image in testdata.
func TestSamples(t *testing.T) {
	for _, f := range faces {
		face := f.face()
		width := bitmap.Measure(face, f.sample)
		img := image.NewGray(image.Rect(0, 0, width+4, 16))
		for i := range img.Pix {
			img.Pix[i] = 0xFF
		}
		bitmap.Draw(img, image.Pt(2, 12), face, f.sample, color.Black)
		var got bytes.Buffer
		if err := png.Encode(&got, img); err != nil {
			t.Fatal(err)
		}
		path := "testdata/" + f.name + ".png"
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
		w, err := png.Decode(bytes.NewReader(want))
		if err != nil {
			t.Fatal(err)
		}
		if w.Bounds() != img.Bounds() {
			t.Fatalf("%s: the sample is %v, its golden image %v", f.name, img.Bounds(), w.Bounds())
		}
		for y := 0; y < img.Bounds().Dy(); y++ {
			for x := 0; x < img.Bounds().Dx(); x++ {
				if color.GrayModel.Convert(w.At(x, y)).(color.Gray) != img.GrayAt(x, y) {
					t.Fatalf("%s differs from %s at %d, %d", f.name, path, x, y)
				}
			}
		}
	}
}
