package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"

	"github.com/firede/platinum-fonts/bitmap"
	"github.com/firede/platinum-fonts/internal/cells"
	"github.com/firede/platinum-fonts/internal/grid"
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
// and Platinum Mono in a terminal, as a PNG.
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
		text(small, "Bitmap faces for Latin and East Asian scripts, on a 12px grid", m, y, label, 1)

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

		section("Mono: Regular and Bold")
		monoText := bitmap.Fallback(mono.regular, monoEastAsian{regular["zhhans"]})
		monoBold := bitmap.Fallback(mono.bold, monoEastAsian{bold["zhhans"]})
		y += 6
		for i, line := range monoBox("main.go", 40, "func main() {", `    fmt.Println("你好，世界") // 问候`, "}") {
			text(monoText, line, m, y+grid.Ascent+grid.H*i, ink, 1)
		}
		x := m + 46*grid.W
		for i, line := range []string{"", "▁▂▃▄▅▆▇█▇▆", "⣀⣠⣤⣴⣶⣾⣿⣷⣦⣄", "░░▒▒▓▓██▓▒", ""} {
			text(monoText, line, x, y+grid.Ascent+grid.H*i, ink, 1)
		}
		y += 5*grid.H + 8
		// A prompt of Powerline segments, in Bold.
		x = m
		segments := []struct {
			s      string
			fg, bg color.Color
		}{{" ~/fonts ", paper, ink}, {" main ", ink, barFace}}
		for i, seg := range segments {
			w := bitmap.Measure(monoBold, seg.s)
			rect(x, y, w, grid.H, seg.bg)
			text(monoBold, seg.s, x, y+grid.Ascent, seg.fg, 1)
			x += w
			next := color.Color(paper)
			if i+1 < len(segments) {
				next = segments[i+1].bg
			}
			rect(x, y, grid.W, grid.H, next)
			x = text(monoBold, "\uE0B0", x, y+grid.Ascent, seg.bg, 1)
		}
		text(monoBold, " go test ./... ", x, y+grid.Ascent, ink, 1)
		y += grid.H
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

// monoBox returns lines of a box drawn with box drawing characters around
// lines of text inner cells wide, its title on its top.
func monoBox(title string, inner int, lines ...string) []string {
	width := func(s string) int {
		n := 0
		for _, r := range s {
			n += cells.Of(r)
		}
		return n
	}
	out := []string{"╭─ " + title + " " + strings.Repeat("─", inner-width(title)-1) + "╮"}
	for _, l := range lines {
		out = append(out, "│ "+l+strings.Repeat(" ", inner-width(l))+" │")
	}
	return append(out, "╰"+strings.Repeat("─", inner+2)+"╯")
}

// monoEastAsian is a face of cjk12 as mono.EastAsian sets it, which the
// build cannot import: a row lower, each glyph advancing the cells of its
// character, the last column of them clear, those that do not fit them
// left out.
type monoEastAsian struct{ f *bitmap.Face }

func (e monoEastAsian) Glyph(r rune) (bitmap.Glyph, bool) {
	g, ok := e.f.Glyph(r)
	if !ok {
		return bitmap.Glyph{}, false
	}
	g = g.Add(image.Pt(0, 1))
	g.Advance = cells.Of(r) * grid.W
	if !g.Bounds.In(image.Rect(0, -grid.Ascent, g.Advance, grid.H-grid.Ascent)) {
		return bitmap.Glyph{}, false
	}
	// cjk12's Bold fills the last column; keep it clear, as Regular does,
	// to keep characters apart.
	if g.Bounds.Max.X == g.Advance && g.Advance > 0 {
		box := g.Bounds
		box.Max.X--
		g = bitmap.NewGlyph(g.Advance, box, g.At)
	}
	return g, true
}
