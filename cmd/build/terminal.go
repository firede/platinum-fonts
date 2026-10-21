package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"

	"github.com/firede/platinum-fonts/bitmap"
	"github.com/firede/platinum-fonts/internal/cells"
	"github.com/firede/platinum-fonts/internal/grid"
	"github.com/firede/platinum-fonts/internal/ligature"
)

// span is a run of text in a terminal, in one style.
type span struct {
	s      string
	bold   bool
	fg, bg color.Color
}

// The terminal of the Mono specimen: its width in cells, and its margin
// within its window, in pixels.
const termCols, termPad = 66, 8

// termText returns the lines of the Mono specimen: a shell session that
// lists a directory with icons, shows code with ligatures and Chinese
// beside a system monitor, and runs the tests.
func termText() [][]span {
	plain := func(s string) span { return span{s: s, fg: ink} }
	bold := func(s string) span { return span{s: s, bold: true, fg: ink} }
	dim := func(s string) span { return span{s: s, fg: label} }
	prompt := func(cmd string) []span {
		return []span{
			{"  ~/platinum-fonts ", true, paper, ink},
			{"", false, ink, barFace},
			{"  main ", true, ink, barFace},
			{"", false, barFace, nil},
			plain(" " + cmd),
		}
	}
	// An entry of eza's grid: an icon, a space and the name, in a column
	// of 16 cells; directories in bold.
	entry := func(icon, name string, dir bool) span {
		return span{s: icon + " " + name + strings.Repeat(" ", 14-textCells(name)), bold: dir, fg: ink}
	}
	folder := func(name string) span { return entry("", name, true) }

	code := []struct{ kw, rest, comment string }{
		{"func", " greet(name string) string {", ""},
		{"    if", ` name != "" && len(name) <= 8 {`, ""},
		{"", `        msg := "你好，" + name `, "// 问候"},
		{"        return", " msg", ""},
		{"", "    }", ""},
		{"    return", ` "Hello, " + name`, ""},
		{"", "}", ""},
	}
	monitor := []struct{ name, bar string }{
		{"CPU", "⣀⣠⣴⣾⣿⣷⣦⣄ 42%"},
		{"内存", "████▌    61%"},
		{"网络", "▁▂▃▅▇▆▃▁ 3MB"},
		{"磁盘", "▓▓▓▒▒░░░ 37%"},
	}
	const codeW, monW = 40, 16
	box := func(title string, w int) (top, bottom span) {
		return dim("╭─ " + title + " " + strings.Repeat("─", w-textCells(title)-1) + "╮"),
			dim("╰" + strings.Repeat("─", w+2) + "╯")
	}
	codeTop, codeBottom := box("hello.go", codeW)
	monTop, monBottom := box("系统", monW)

	lines := [][]span{
		prompt("eza"),
		{folder("bitmap"), folder("cjk12"), folder("cmd"), folder("docs")},
		{folder("glyphs"), folder("mono"), folder("sans"), entry("", "go.mod", false)},
		{entry("", "LICENSE", false), entry("", "NOTICE.md", false), entry("\U000F00BA", "README.md", false), entry("", "sources.toml", false)},
		nil,
		prompt("bat hello.go"),
		{codeTop, plain("  "), monTop},
	}
	for i, c := range code {
		l := []span{dim("│"), dim(" " + string(rune('1'+i)) + " ")}
		if c.kw != "" {
			l = append(l, bold(c.kw))
		}
		l = append(l, plain(c.rest))
		if c.comment != "" {
			l = append(l, dim(c.comment))
		}
		pad := codeW - 2 - textCells(c.kw+c.rest+c.comment)
		l = append(l, plain(strings.Repeat(" ", pad)), dim(" │"), plain("  "))
		switch {
		case i < len(monitor):
			m := monitor[i]
			l = append(l, dim("│ "), bold(m.name+strings.Repeat(" ", 4-textCells(m.name))), plain(m.bar), dim(" │"))
		case i == len(monitor):
			l = append(l, monBottom)
		}
		lines = append(lines, l)
	}
	return append(lines,
		[]span{codeBottom},
		nil,
		prompt("go test ./..."),
		[]span{plain("ok   github.com/firede/platinum-fonts/mono   0.87s "), bold("")},
		append(prompt(""), span{s: "█", fg: ink}),
	)
}

// monoSpecimen draws Platinum Mono in a terminal, in a window of the
// Platinum appearance, as a PNG.
func monoSpecimen(latin []latinFace, cjk *cjkMade, mono *monoMade) ([]byte, error) {
	var title *bitmap.Face
	for _, l := range latin {
		if l.family == "Platinum Sans" && l.style == "Bold" {
			title = l.face
		}
	}
	regularEA, boldEA, err := regionFaces(cjk, "zhhans")
	if err != nil {
		return nil, err
	}
	faces := map[bool]bitmap.Font{
		false: bitmap.Fallback(mono.regular, monoEastAsian{regularEA}),
		true:  bitmap.Fallback(mono.bold, monoEastAsian{boldEA}),
	}
	ligs := map[bool][][]bitmap.Glyph{false: mono.ligRegular, true: mono.ligBold}

	const (
		k      = specimenScale
		margin = 6 // around the window, for its shadow
		barH   = 19
	)
	lines := termText()
	winW := termCols*grid.W + 2*termPad + 2
	winH := barH + 1 + len(lines)*grid.H + 2*termPad + 1
	w, h := winW+2*margin+1, winH+2*margin+1
	img := image.NewRGBA(image.Rect(0, 0, w*k, h*k))
	px := scaled{img, k}
	rect := func(x, y, w, h int, c color.Color) {
		for j := y; j < y+h; j++ {
			for i := x; i < x+w; i++ {
				px.Set(i, j, c)
			}
		}
	}
	rect(0, 0, w, h, paper)

	// The window: a shadow, a frame, and a title bar of pinstripes with a
	// close box and the title.
	x0, y0 := margin, margin
	rect(x0+1, y0+1, winW, winH, ink)
	rect(x0, y0, winW, winH, ink)
	rect(x0+1, y0+1, winW-2, barH-1, barFace)
	for y := y0 + 4; y < y0+barH-3; y += 2 {
		rect(x0+4, y, winW-8, 1, rule)
	}
	rect(x0+9, y0+4, 11, 11, ink)
	rect(x0+10, y0+5, 9, 9, barFace)
	name := "Terminal — Platinum Mono"
	tw := bitmap.Measure(title, name)
	rect(x0+(winW-tw)/2-6, y0+3, tw+12, barH-5, barFace)
	bitmap.Draw(px, image.Pt(x0+(winW-tw)/2, y0+14), title, name, ink)
	rect(x0, y0+barH, winW, 1, ink)
	rect(x0+1, y0+barH+1, winW-2, winH-barH-2, paper)

	// The terminal: the backgrounds of a line first, then its glyphs, as
	// an icon draws into the cell after it.
	tx, ty := x0+1+termPad, y0+barH+1+termPad
	for row, line := range lines {
		top := ty + row*grid.H
		x := tx
		for _, sp := range line {
			n := textCells(sp.s)
			if sp.bg != nil {
				rect(x, top, n*grid.W, grid.H, sp.bg)
			}
			x += n * grid.W
		}
		dot := image.Pt(tx, top+grid.Ascent)
		for _, sp := range line {
			rs := []rune(sp.s)
			starts := ligature.Starts(mono.ligTexts, rs)
			for i := 0; i < len(rs); i++ {
				if j := starts[i]; j >= 0 {
					for _, g := range ligs[sp.bold][j] {
						dot = bitmap.DrawGlyph(px, dot, g, sp.fg)
					}
					i += len(ligs[sp.bold][j]) - 1
				} else if g, ok := faces[sp.bold].Glyph(rs[i]); ok {
					dot = bitmap.DrawGlyph(px, dot, g, sp.fg)
				} else {
					dot.X += cells.Of(rs[i]) * grid.W
				}
			}
		}
	}

	var b bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&b, img); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// textCells returns the cells s takes in a terminal.
func textCells(s string) int {
	n := 0
	for _, r := range s {
		n += cells.Of(r)
	}
	return n
}
