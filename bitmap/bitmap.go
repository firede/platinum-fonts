// Package bitmap holds a bitmap face: one bit per pixel, a glyph for each of
// its characters, packed in a compact form that Encode writes and Decode
// reads.
//
// Coordinates are in pixels from the dot, the point on the baseline where a
// glyph is drawn, with y growing downward: a glyph whose ink rises ten rows
// over the baseline and goes one under it has bounds from y = -10 to y = 1.
package bitmap

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"slices"
	"sort"
)

// Glyph is the drawing of a character.
type Glyph struct {
	// Advance is how far the dot moves after the glyph.
	Advance int
	// Bounds is the box of the glyph's pixels, from the dot.
	Bounds image.Rectangle
	bits   []byte // Bounds' pixels, row by row, a bit each, the high bit first
}

// NewGlyph returns a glyph advancing advance, whose pixels in bounds are
// ink where ink reports so.
func NewGlyph(advance int, bounds image.Rectangle, ink func(x, y int) bool) Glyph {
	g := Glyph{Advance: advance, Bounds: bounds.Canon()}
	g.bits = make([]byte, bitBytes(g.Bounds.Dx(), g.Bounds.Dy()))
	i := 0
	for y := g.Bounds.Min.Y; y < g.Bounds.Max.Y; y++ {
		for x := g.Bounds.Min.X; x < g.Bounds.Max.X; x++ {
			if ink(x, y) {
				g.bits[i/8] |= 0x80 >> (i % 8)
			}
			i++
		}
	}
	return g
}

// At reports whether the pixel at x, y, from the dot, is ink.
func (g Glyph) At(x, y int) bool {
	if !image.Pt(x, y).In(g.Bounds) {
		return false
	}
	i := (y-g.Bounds.Min.Y)*g.Bounds.Dx() + x - g.Bounds.Min.X
	return g.bits[i/8]&(0x80>>(i%8)) != 0
}

// Mask returns the glyph as an alpha mask whose bounds are the glyph's,
// from the dot: opaque where it is ink.
func (g Glyph) Mask() *image.Alpha {
	m := image.NewAlpha(g.Bounds)
	for y := g.Bounds.Min.Y; y < g.Bounds.Max.Y; y++ {
		for x := g.Bounds.Min.X; x < g.Bounds.Max.X; x++ {
			if g.At(x, y) {
				m.Pix[m.PixOffset(x, y)] = 0xFF
			}
		}
	}
	return m
}

func bitBytes(w, h int) int { return (w*h + 7) / 8 }

// metrics is what glyphs share: their advance and bounds.
type metrics struct {
	advance    int8
	x, y, w, h int8
}

func (m metrics) bounds() image.Rectangle {
	return image.Rect(int(m.x), int(m.y), int(m.x)+int(m.w), int(m.y)+int(m.h))
}

type span struct {
	start rune
	first int // the index of its first glyph
	n     int
}

// Face is a decoded face.
type Face struct {
	ascent, descent int
	spans           []span
	metrics         []metrics
	glyphMetrics    []uint16 // by glyph index
	offsets         []uint32 // of each glyph's bits; one more for the end
	bits            []byte
}

// Ascent returns the rows of the face over the baseline.
func (f *Face) Ascent() int { return f.ascent }

// Descent returns the rows of the face under the baseline.
func (f *Face) Descent() int { return f.descent }

// Len returns the number of characters the face draws.
func (f *Face) Len() int { return len(f.glyphMetrics) }

func (f *Face) index(r rune) (int, bool) {
	i := sort.Search(len(f.spans), func(i int) bool { return f.spans[i].start > r }) - 1
	if i < 0 || r >= f.spans[i].start+rune(f.spans[i].n) {
		return 0, false
	}
	return f.spans[i].first + int(r-f.spans[i].start), true
}

// Has reports whether the face draws r.
func (f *Face) Has(r rune) bool {
	_, ok := f.index(r)
	return ok
}

// Glyph returns the glyph of r, and whether the face has one.
func (f *Face) Glyph(r rune) (Glyph, bool) {
	i, ok := f.index(r)
	if !ok {
		return Glyph{}, false
	}
	m := f.metrics[f.glyphMetrics[i]]
	return Glyph{
		Advance: int(m.advance),
		Bounds:  m.bounds(),
		bits:    f.bits[f.offsets[i]:f.offsets[i+1]:f.offsets[i+1]],
	}, true
}

// Runes calls fn with every character of the face, in order, until fn
// returns false.
func (f *Face) Runes(fn func(r rune) bool) {
	for _, s := range f.spans {
		for i := range s.n {
			if !fn(s.start + rune(i)) {
				return
			}
		}
	}
}

// The encoded form, all little endian:
//
//	"PXF1"
//	ascent, descent        int16 each
//	metrics count          uint16
//	metrics                advance, x, y, w, h: int8 each
//	spans count            uint32
//	spans                  first rune, glyph count: uint32 each
//	glyphs' metrics        uint16 each, in the order of the spans
//	glyphs' bits           row by row, high bit first, each glyph from a
//	                       byte of its own
const magic = "PXF1"

// Encode packs a face of ascent and descent with glyphs, by character.
func Encode(ascent, descent int, glyphs map[rune]Glyph) ([]byte, error) {
	runes := make([]rune, 0, len(glyphs))
	for r := range glyphs {
		runes = append(runes, r)
	}
	slices.Sort(runes)
	var ms []metrics
	index := map[metrics]uint16{}
	b := []byte(magic)
	b = binary.LittleEndian.AppendUint16(b, uint16(int16(ascent)))
	b = binary.LittleEndian.AppendUint16(b, uint16(int16(descent)))
	var spans []span
	var glyphMetrics []uint16
	var bits []byte
	for i, r := range runes {
		g := glyphs[r]
		bd := g.Bounds
		if g.Advance < -128 || g.Advance > 127 || bd.Min.X < -128 || bd.Min.X > 127 ||
			bd.Min.Y < -128 || bd.Min.Y > 127 || bd.Dx() > 127 || bd.Dy() > 127 {
			return nil, fmt.Errorf("bitmap: the glyph of %U is too large", r)
		}
		m := metrics{int8(g.Advance), int8(bd.Min.X), int8(bd.Min.Y), int8(bd.Dx()), int8(bd.Dy())}
		k, ok := index[m]
		if !ok {
			if len(ms) == 1<<16 {
				return nil, errors.New("bitmap: too many kinds of glyph")
			}
			k = uint16(len(ms))
			index[m] = k
			ms = append(ms, m)
		}
		glyphMetrics = append(glyphMetrics, k)
		if len(g.bits) != bitBytes(bd.Dx(), bd.Dy()) {
			return nil, fmt.Errorf("bitmap: the glyph of %U has %d bytes, not %d", r, len(g.bits), bitBytes(bd.Dx(), bd.Dy()))
		}
		bits = append(bits, g.bits...)
		if i > 0 && r == runes[i-1]+1 {
			spans[len(spans)-1].n++
		} else {
			spans = append(spans, span{start: r, first: i, n: 1})
		}
	}
	b = binary.LittleEndian.AppendUint16(b, uint16(len(ms)))
	for _, m := range ms {
		b = append(b, byte(m.advance), byte(m.x), byte(m.y), byte(m.w), byte(m.h))
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(spans)))
	for _, s := range spans {
		b = binary.LittleEndian.AppendUint32(b, uint32(s.start))
		b = binary.LittleEndian.AppendUint32(b, uint32(s.n))
	}
	for _, k := range glyphMetrics {
		b = binary.LittleEndian.AppendUint16(b, k)
	}
	return append(b, bits...), nil
}

var errCorrupt = errors.New("bitmap: corrupt face")

// Decode reads a face Encode packed. The face keeps data.
func Decode(data []byte) (*Face, error) {
	if len(data) < 10 || string(data[:4]) != magic {
		return nil, errors.New("bitmap: not a face")
	}
	f := &Face{
		ascent:  int(int16(binary.LittleEndian.Uint16(data[4:]))),
		descent: int(int16(binary.LittleEndian.Uint16(data[6:]))),
	}
	nm := int(binary.LittleEndian.Uint16(data[8:]))
	p := 10
	if len(data) < p+5*nm+4 {
		return nil, errCorrupt
	}
	f.metrics = make([]metrics, nm)
	for i := range f.metrics {
		d := data[p:]
		f.metrics[i] = metrics{int8(d[0]), int8(d[1]), int8(d[2]), int8(d[3]), int8(d[4])}
		if f.metrics[i].w < 0 || f.metrics[i].h < 0 {
			return nil, errCorrupt
		}
		p += 5
	}
	ns := int(binary.LittleEndian.Uint32(data[p:]))
	p += 4
	if ns < 0 || len(data)-p < 8*ns {
		return nil, errCorrupt
	}
	f.spans = make([]span, ns)
	n := 0
	for i := range f.spans {
		start := rune(binary.LittleEndian.Uint32(data[p:]))
		count := int(binary.LittleEndian.Uint32(data[p+4:]))
		if count <= 0 || count > 1<<21 || i > 0 && start < f.spans[i-1].start+rune(f.spans[i-1].n) {
			return nil, errCorrupt
		}
		f.spans[i] = span{start: start, first: n, n: count}
		n += count
		p += 8
	}
	if len(data)-p < 2*n {
		return nil, errCorrupt
	}
	f.glyphMetrics = make([]uint16, n)
	f.offsets = make([]uint32, n+1)
	off := 0
	for i := range f.glyphMetrics {
		k := binary.LittleEndian.Uint16(data[p:])
		if int(k) >= nm {
			return nil, errCorrupt
		}
		f.glyphMetrics[i] = k
		f.offsets[i] = uint32(off)
		off += bitBytes(int(f.metrics[k].w), int(f.metrics[k].h))
		p += 2
	}
	f.offsets[n] = uint32(off)
	if len(data)-p != off {
		return nil, errCorrupt
	}
	f.bits = data[p:]
	return f, nil
}
