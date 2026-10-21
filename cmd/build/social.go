package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"github.com/firede/platinum-fonts/bitmap"
	"github.com/firede/platinum-fonts/internal/cells"
	"github.com/firede/platinum-fonts/internal/grid"
	"github.com/firede/platinum-fonts/internal/ligature"
)

// The social preview's size, in pixels of the image, as GitHub asks for
// it, and the scale of its window: 320×160 pixels of the faces.
const socialW, socialH, socialScale = 1280, 640, 4

var desk = color.RGBA{0xCC, 0xCC, 0xCC, 0xFF}

// socialPreview draws the repository's social preview, as a PNG: a window
// of the Platinum appearance on a desktop, holding a greeting in each
// script in Sans Bold, the tagline, a prompt of Platinum Mono with icons
// and ligatures, and the names of the families, each in itself, and the
// regions.
func socialPreview(latin []latinFace, cjk *cjkMade, mono *monoMade) ([]byte, error) {
	face := func(family, style string) *bitmap.Face {
		for _, l := range latin {
			if l.family == family && l.style == style {
				return l.face
			}
		}
		panic("no face " + family + " " + style)
	}
	title := face("Platinum Sans", "Bold")
	hero := bitmap.Font(title)
	for _, pkg := range []string{"zhhans", "ja", "ko"} {
		_, bold, err := regionFaces(cjk, pkg)
		if err != nil {
			return nil, err
		}
		hero = bitmap.Fallback(hero, bold)
	}
	regularEA, boldEA, err := regionFaces(cjk, "zhhans")
	if err != nil {
		return nil, err
	}
	monoFaces := map[bool]bitmap.Font{
		false: bitmap.Fallback(mono.regular, monoEastAsian{regularEA}),
		true:  bitmap.Fallback(mono.bold, monoEastAsian{boldEA}),
	}
	ligs := map[bool][][]bitmap.Glyph{false: mono.ligRegular, true: mono.ligBold}

	img := image.NewRGBA(image.Rect(0, 0, socialW, socialH))
	// rect fills a rectangle in pixels of the image scaled k times.
	rect := func(k, x, y, w, h int, c color.Color) {
		for j := y * k; j < (y+h)*k; j++ {
			for i := x * k; i < (x+w)*k; i++ {
				img.Set(i, j, c)
			}
		}
	}
	// text draws s at k times, x and y in pixels of the image, multiples
	// of k; it returns the x after s.
	text := func(f bitmap.Font, s string, x, y, k int, c color.Color) int {
		return bitmap.Draw(scaled{img, k}, image.Pt(x/k, y/k), f, s, c).X * k
	}
	// monoText draws spans of Platinum Mono at k times, their backgrounds
	// first, as an icon draws into the cell after it.
	monoText := func(line []span, x, y, k int) {
		px := scaled{img, k}
		cx := x / k
		for _, sp := range line {
			n := textCells(sp.s)
			if sp.bg != nil {
				rect(k, cx, y/k, n*grid.W, grid.H, sp.bg)
			}
			cx += n * grid.W
		}
		dot := image.Pt(x/k, y/k+grid.Ascent)
		for _, sp := range line {
			rs := []rune(sp.s)
			starts := ligature.Starts(mono.ligTexts, rs)
			for i := 0; i < len(rs); i++ {
				if j := starts[i]; j >= 0 {
					for _, g := range ligs[sp.bold][j] {
						dot = bitmap.DrawGlyph(px, dot, g, sp.fg)
					}
					i += len(ligs[sp.bold][j]) - 1
				} else if g, ok := monoFaces[sp.bold].Glyph(rs[i]); ok {
					dot = bitmap.DrawGlyph(px, dot, g, sp.fg)
				} else {
					dot.X += cells.Of(rs[i]) * grid.W
				}
			}
		}
	}

	// The desktop and the window, as in the Mono specimen, at
	// socialScale.
	const (
		k                  = socialScale
		wx, wy, winW, winH = 12, 10, 296, 138
		barH               = 19
	)
	rect(1, 0, 0, socialW, socialH, desk)
	rect(k, wx+1, wy+1, winW, winH, ink)
	rect(k, wx, wy, winW, winH, ink)
	rect(k, wx+1, wy+1, winW-2, barH-1, barFace)
	for y := wy + 4; y < wy+barH-3; y += 2 {
		rect(k, wx+4, y, winW-8, 1, rule)
	}
	rect(k, wx+9, wy+4, 11, 11, ink)
	rect(k, wx+10, wy+5, 9, 9, barFace)
	name := "Platinum Fonts"
	tw := bitmap.Measure(title, name)
	rect(k, wx+(winW-tw)/2-6, wy+3, tw+12, barH-5, barFace)
	bitmap.Draw(scaled{img, k}, image.Pt(wx+(winW-tw)/2, wy+14), title, name, ink)
	rect(k, wx, wy+barH, winW, 1, ink)
	rect(k, wx+1, wy+barH+1, winW-2, winH-barH-2, paper)

	// The contents, in pixels of the image.
	left, right := (wx+1+12)*k, (wx+winW-1-12)*k
	y := (wy+barH+1)*k + 132
	text(hero, "Hello 你好 こんにちは 안녕", left, y, 6, ink)
	y += 64
	text(face("Platinum Sans", "Regular"), "Pixel fonts for interfaces and terminals, on a 12-pixel grid", left, y, 3, label)
	y += 48
	monoText([]span{
		{"  ~/platinum-fonts ", true, paper, ink},
		{"", false, ink, barFace},
		{"  main ", true, ink, barFace},
		{"", false, barFace, nil},
		{` a != b => "你好"`, false, ink, nil},
	}, left, y, 3)

	y = (wy+winH-1)*k - 104
	rect(1, left, y, right-left, k, rule)
	y += 64
	x := text(title, "Sans", left, y, 3, ink) + 48
	x = text(face("Platinum Sans Small", "Bold"), "Sans Small", x, y, 3, ink) + 48
	monoText([]span{{s: "Mono", bold: true, fg: ink}}, x, y-grid.Ascent*3, 3)
	regions := "SC  TC  JP  KR"
	text(title, regions, right-bitmap.Measure(title, regions)*3, y, 3, label)

	var b bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&b, img); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
