package main

import (
	"fmt"
	"image"
	"math"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"

	"github.com/firede/platinum-fonts/bitmap"
	"github.com/firede/platinum-fonts/internal/cells"
	"github.com/firede/platinum-fonts/internal/drawn"
	"github.com/firede/platinum-fonts/internal/grid"
	"github.com/firede/platinum-fonts/internal/source"
)

// iconsDrawn is the drawing of the icons Platinum Mono does not take as
// they rasterize.
const iconsDrawn = "glyphs/icons.txt"

// iconSets are the icon sets Nerd Fonts gathers, as it lists them, and the
// files buildIcons writes their licenses to, from the sources named.
var iconSets = []struct {
	name, url, license string
	sources            []string
}{
	{"Codicons", "https://github.com/microsoft/vscode-codicons", "CC BY 4.0", []string{"nerd-fonts-license-codicons"}},
	{"Devicons", "https://github.com/devicons/devicon", "MIT", nil},
	{"extraglyphs", "https://github.com/source-foundry/Hack", "MIT", nil},
	{"Font Awesome", "https://github.com/FortAwesome/Font-Awesome", "CC BY 4.0", []string{"nerd-fonts-license-font-awesome"}},
	{"Font Awesome Extension", "https://github.com/AndreLZGava/font-awesome-extension", "MIT", nil},
	{"Font Logos", "https://github.com/lukas-w/font-logos", "unlicensed", nil},
	{"MaterialDesign", "https://github.com/Templarian/MaterialDesign-Font", "Apache 2.0", []string{"nerd-fonts-license-materialdesign", "apache-license"}},
	{"Octicons", "https://github.com/primer/octicons", "MIT", []string{"nerd-fonts-license-octicons"}},
	{"Seti and original", "https://github.com/jesseweed/seti-ui", "MIT", nil},
	{"Pomicons", "https://github.com/gabrielelana/pomicons", "OFL 1.1 RFN", []string{"nerd-fonts-license-pomicons"}},
	{"Powerline Extra", "https://github.com/ryanoasis/powerline-extra-symbols", "MIT", []string{"nerd-fonts-license-powerline-extra"}},
	{"Powerline Symbols", "https://github.com/powerline/powerline", "MIT", []string{"nerd-fonts-license-powerline-symbols"}},
	{"Power Symbols IEC", "https://github.com/jloughry/Unicode", "MIT", nil},
	{"Weather Icons", "https://github.com/erikflowers/weather-icons", "OFL 1.1", []string{"nerd-fonts-license-weather-icons"}},
}

// licenseFile returns where buildIcons writes the license of a source.
func licenseFile(name string) string {
	switch name {
	case "nerd-fonts-symbols":
		return "licenses/nerd-fonts/LICENSE.txt"
	case "apache-license":
		return "licenses/nerd-fonts/apache-2.0.txt"
	}
	return "licenses/nerd-fonts/" + name[len("nerd-fonts-license-"):] + ".txt"
}

// iconsMade is what buildIcons made, for the docs.
type iconsMade struct {
	src          *source.Source
	icons, drawn int
}

// buildIcons returns the icons of Nerd Fonts for Platinum Mono: the glyphs
// of Symbols Nerd Font in the Private Use Area that Mono has none of. Each
// takes a cell and is drawn up to two cells wide, as the Nerd Font (not
// Nerd Font Mono) fonts set them, so a space should follow it. They are
// rasterized with the font's line on Mono's cells, centred on two cells
// but for the Powerline symbols, which keep their place; those drawn in
// glyphs/icons.txt replace them.
func buildIcons(sources map[string]*source.Source, cache string, have map[rune]bitmap.Glyph, out map[string][]byte) (map[rune]bitmap.Glyph, *iconsMade, error) {
	fetch := func(name string) (*source.Source, []byte, error) {
		src := sources[name]
		if src == nil {
			return nil, nil, fmt.Errorf("sources.toml has no %s", name)
		}
		data, err := source.Fetch(src, cache)
		return src, data, err
	}
	src, data, err := fetch("nerd-fonts-symbols")
	if err != nil {
		return nil, nil, err
	}
	archive, err := source.OpenArchive(data)
	if err != nil {
		return nil, nil, err
	}
	if out[licenseFile(src.Name)], err = archive.File("LICENSE"); err != nil {
		return nil, nil, err
	}
	for _, set := range iconSets {
		for _, name := range set.sources {
			if _, out[licenseFile(name)], err = fetch(name); err != nil {
				return nil, nil, err
			}
		}
	}
	ttf, err := archive.File("SymbolsNerdFont-Regular.ttf")
	if err != nil {
		return nil, nil, err
	}
	f, err := sfnt.Parse(ttf)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %v", src.Name, err)
	}

	var b sfnt.Buffer
	upm := fixed.I(int(f.UnitsPerEm()))
	m, err := f.Metrics(&b, upm, font.HintingNone)
	if err != nil {
		return nil, nil, err
	}
	// Pixels a unit: the font's line on the cell's height.
	scale := float64(grid.H) / (float64(m.Ascent+m.Descent) / 64)
	icons := map[rune]bitmap.Glyph{}
	for _, rs := range [][2]rune{{0xE000, 0xF8FF}, {0xF0000, 0xFFFFD}, {0x100000, 0x10FFFD}} {
		for r := rs[0]; r <= rs[1]; r++ {
			if _, ok := have[r]; ok {
				continue
			}
			gi, err := f.GlyphIndex(&b, r)
			if err != nil {
				return nil, nil, err
			}
			if gi == 0 {
				continue
			}
			segs, err := f.LoadGlyph(&b, gi, upm, nil)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %U: %v", src.Name, r, err)
			}
			if g, ok := rasterize(segs, scale, !powerline(r)); ok {
				icons[r] = g
			}
		}
	}

	text, err := os.ReadFile(iconsDrawn)
	if err != nil {
		return nil, nil, err
	}
	own, err := drawn.Parse(iconsDrawn, string(text))
	if err != nil {
		return nil, nil, err
	}
	if own.Height != grid.H || own.Ascent != grid.Ascent {
		return nil, nil, fmt.Errorf("%s: not drawn on the grid: height %d, ascent %d", iconsDrawn, own.Height, own.Ascent)
	}
	for r, g := range own.Glyphs {
		if _, ok := icons[r]; !ok {
			return nil, nil, fmt.Errorf("%s: %U is not an icon of %s", iconsDrawn, r, src.Name)
		}
		ink := inkBox(g)
		if !ink.In(iconBox) {
			return nil, nil, fmt.Errorf("%s: %U is not drawn in %v", iconsDrawn, r, iconBox)
		}
		icons[r] = bitmap.NewGlyph(grid.W, ink, g.At)
	}
	for r := range icons {
		if cells.Of(r) != 1 {
			return nil, nil, fmt.Errorf("%U takes %d cells", r, cells.Of(r))
		}
	}
	return icons, &iconsMade{src: src, icons: len(icons), drawn: len(own.Glyphs)}, nil
}

// iconBox is where an icon may be drawn: two cells, the first column of
// the first left clear between it and the character before.
var iconBox = image.Rect(1, -grid.Ascent, 2*grid.W, grid.H-grid.Ascent)

// powerline reports whether r is a Powerline symbol, which stands in its
// place on the line rather than in the middle of two cells.
func powerline(r rune) bool { return r >= 0xE0A0 && r <= 0xE0D7 }

// rasterize returns the glyph of segs, in the font's units with y down, at
// scale pixels a unit, its ink centred on two cells if centre. It returns
// false if no pixel is inked.
func rasterize(segs sfnt.Segments, scale float64, centre bool) (bitmap.Glyph, bool) {
	ox := 0.0
	if centre {
		bd := segs.Bounds()
		ox = math.Round(float64(grid.W) - float64(bd.Min.X+bd.Max.X)/128*scale)
	}
	box := image.Rect(0, -grid.Ascent, 2*grid.W, grid.H-grid.Ascent)
	z := vector.NewRasterizer(box.Dx(), box.Dy())
	p := func(q fixed.Point26_6) (float32, float32) {
		return float32(float64(q.X)/64*scale + ox), float32(float64(q.Y)/64*scale + grid.Ascent)
	}
	for _, s := range segs {
		a := s.Args
		switch s.Op {
		case sfnt.SegmentOpMoveTo:
			z.MoveTo(p(a[0]))
		case sfnt.SegmentOpLineTo:
			z.LineTo(p(a[0]))
		case sfnt.SegmentOpQuadTo:
			x1, y1 := p(a[0])
			x2, y2 := p(a[1])
			z.QuadTo(x1, y1, x2, y2)
		case sfnt.SegmentOpCubeTo:
			x1, y1 := p(a[0])
			x2, y2 := p(a[1])
			x3, y3 := p(a[2])
			z.CubeTo(x1, y1, x2, y2, x3, y3)
		}
	}
	z.ClosePath()
	cover := image.NewAlpha(image.Rect(0, 0, box.Dx(), box.Dy()))
	z.Draw(cover, cover.Bounds(), image.Opaque, image.Point{})
	// A pixel is inked where the outline covers half of it, or less where
	// that would lose a quarter of the ink, as thin lines do.
	var sum int
	count := map[uint8]int{}
	for _, a := range cover.Pix {
		sum += int(a)
		count[a]++
	}
	threshold := uint8(0x80)
	for _, t := range []uint8{0x80, 0x60, 0x40} {
		threshold = t
		n := 0
		for a, c := range count {
			if a >= t {
				n += c
			}
		}
		if n*0xFF*4 >= sum*3 {
			break
		}
	}
	ink := func(x, y int) bool {
		return image.Pt(x, y).In(iconBox) && cover.AlphaAt(x, y+grid.Ascent).A >= threshold
	}
	g := bitmap.NewGlyph(grid.W, box, ink)
	tight := inkBox(g)
	if tight.Empty() {
		return bitmap.Glyph{}, false
	}
	return bitmap.NewGlyph(grid.W, tight, ink), true
}
