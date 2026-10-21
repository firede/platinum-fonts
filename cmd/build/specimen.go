package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"

	"github.com/firede/platinum-fonts/bitmap"
	"github.com/firede/platinum-fonts/internal/cells"
	"github.com/firede/platinum-fonts/internal/grid"
	"github.com/firede/platinum-fonts/internal/ligature"
)

// The specimen's width, in pixels of the faces, and the scale it is drawn
// at: twice, so that a README shows it at about its size, its pixels
// sharp.
const specimenW, specimenScale = 420, 2

var (
	paper   = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	ink     = color.RGBA{0x00, 0x00, 0x00, 0xFF}
	label   = color.RGBA{0x77, 0x77, 0x77, 0xFF}
	rule    = color.RGBA{0xBB, 0xBB, 0xBB, 0xFF}
	barFace = color.RGBA{0xDD, 0xDD, 0xDD, 0xFF}
)

// regions are the specimen's rows of cjk12: its label, a line of words
// whose strokes stand close, and a menu bar's titles.
var regions = []struct {
	pkg, label, words string
	menus             []string
}{
	{"zhhans", "SC", "编辑 显示 藏鑫 覆盖", []string{"文件", "编辑", "显示", "特别", "帮助"}},
	{"zhhant", "TC", "編輯 顯示 繁體 鬱鑿", []string{"檔案", "編輯", "顯示", "特別", "輔助說明"}},
	{"ja", "JP", "編集 表示 書類 鬱", []string{"ファイル", "編集", "表示", "特別", "ヘルプ"}},
	{"ko", "KR", "편집 보기 쳇바퀴", []string{"파일", "편집", "보기", "특수", "도움말"}},
}

// specimen draws the faces: the Latin families in each style, the East
// Asian faces of each region in Regular and Bold, menu bars of their bold,
// and a line of Platinum Mono in each style, as a PNG.
func specimen(latin []latinFace, cjk *cjkMade, mono *monoMade) ([]byte, error) {
	face := func(family, style string) *bitmap.Face {
		for _, l := range latin {
			if l.family == family && l.style == style {
				return l.face
			}
		}
		panic("no face " + family + " " + style)
	}
	title := face("Platinum Sans", "Bold")
	small := face("Platinum Sans Small", "Regular")
	regular, bold := map[string]*bitmap.Face{}, map[string]*bitmap.Face{}
	for _, r := range regions {
		var err error
		if regular[r.pkg], bold[r.pkg], err = regionFaces(cjk, r.pkg); err != nil {
			return nil, err
		}
	}

	// Draw once to measure, then again on an image of the height found.
	draw := func(img *image.RGBA) int {
		k := specimenScale
		text := func(f bitmap.Font, s string, x, y int, c color.Color, size int) int {
			if img == nil {
				return x + bitmap.Measure(f, s)*size
			}
			return bitmap.Draw(scaled{img, k * size}, image.Pt(x/size, y/size), f, s, c).X * size
		}
		rect := func(x, y, w, h int, c color.Color) {
			if img == nil {
				return
			}
			for j := y * k; j < (y+h)*k; j++ {
				for i := x * k; i < (x+w)*k; i++ {
					img.Set(i, j, c)
				}
			}
		}
		const m = 12
		y := m
		section := func(name string) {
			y += 10
			rect(m, y, specimenW-2*m, 1, rule)
			y += 16
			text(title, name, m, y, ink, 1)
			y += 8
		}

		// The name of the family, twice as large.
		y += 18
		text(title, "Platinum Fonts", m, y, ink, 2)
		y += 15
		text(small, "Pixel fonts for interfaces and terminals, on a 12-pixel grid", m, y, label, 1)

		section("Latin")
		for _, l := range latin {
			y += 14
			text(small, l.family+" "+l.style, m, y, label, 1)
			y += 14
			text(l.face, "The quick brown fox jumps over the lazy dog. Ångström, Łódź", m, y, ink, 1)
		}

		section("East Asian: Regular and Bold")
		for _, r := range regions {
			y += 18
			text(title, r.label, m, y, label, 1)
			text(bitmap.Fallback(face("Platinum Sans", "Regular"), regular[r.pkg]), r.words, 44, y, ink, 1)
			text(bitmap.Fallback(title, bold[r.pkg]), r.words, 228, y, ink, 1)
		}

		section("Bold in a menu bar")
		for _, r := range regions {
			menu := bitmap.Fallback(title, bold[r.pkg])
			y += 6
			rect(m, y, specimenW-2*m, 19, barFace)
			rect(m, y+19, specimenW-2*m, 1, ink)
			x := m + 14
			for _, t := range r.menus {
				x = text(menu, t, x, y+14, ink, 1) + 14
			}
			clock := "9:41"
			text(menu, clock, specimenW-m-10-bitmap.Measure(menu, clock), y+14, ink, 1)
			y += 20
		}

		section("Mono")
		monoFaces := []struct {
			style string
			font  bitmap.Font
			ligs  [][]bitmap.Glyph
		}{
			{"Regular", bitmap.Fallback(mono.regular, monoEastAsian{regular["zhhans"]}), mono.ligRegular},
			{"Bold", bitmap.Fallback(mono.bold, monoEastAsian{bold["zhhans"]}), mono.ligBold},
		}
		for _, f := range monoFaces {
			y += 14
			text(small, "Platinum Mono "+f.style, m, y, label, 1)
			y += 14
			if img == nil {
				continue
			}
			rs := []rune(`if a != b && n <= 9 { fmt.Println("你好") } // ✓`)
			starts := ligature.Starts(mono.ligTexts, rs)
			dot := image.Pt(m, y)
			for i := 0; i < len(rs); i++ {
				if j := starts[i]; j >= 0 {
					for _, g := range f.ligs[j] {
						dot = bitmap.DrawGlyph(scaled{img, k}, dot, g, ink)
					}
					i += len(f.ligs[j]) - 1
				} else if g, ok := f.font.Glyph(rs[i]); ok {
					dot = bitmap.DrawGlyph(scaled{img, k}, dot, g, ink)
				}
			}
		}
		return y + m
	}
	h := draw(nil)
	img := image.NewRGBA(image.Rect(0, 0, specimenW*specimenScale, h*specimenScale))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = paper.R, paper.G, paper.B, paper.A
	}
	draw(img)
	var b bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&b, img); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// regionFaces returns the Regular and Bold faces of a package of cjk12,
// Bold made as the package makes it.
func regionFaces(cjk *cjkMade, pkg string) (regular, bold *bitmap.Face, err error) {
	for _, f := range cjk.faces {
		if f.pkg == pkg {
			bold, err := bitmap.EmboldenOpen(f.face)
			return f.face, bold, err
		}
	}
	return nil, nil, fmt.Errorf("no package %s in cjk12", pkg)
}

// monoEastAsian is a face of cjk12 as mono.EastAsian sets it, which the
// build cannot import: a column to the right, each glyph advancing the
// cells of its character, those that do not fit them left out.
type monoEastAsian struct{ f *bitmap.Face }

func (e monoEastAsian) Glyph(r rune) (bitmap.Glyph, bool) {
	g, ok := e.f.Glyph(r)
	if !ok {
		return bitmap.Glyph{}, false
	}
	g = g.Add(image.Pt(1, 0))
	g.Advance = cells.Of(r) * grid.W
	if !g.Bounds.In(image.Rect(0, -grid.Ascent, g.Advance, grid.H-grid.Ascent)) {
		return bitmap.Glyph{}, false
	}
	return g, true
}
