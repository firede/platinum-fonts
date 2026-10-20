// Package grid draws the characters that fill a cell of Platinum Mono and
// join the cells around them: box drawing, block elements, braille
// patterns and Powerline's separators.
//
// A cell is W pixels wide and H tall, Ascent of them over the baseline.
// Light lines run along column 2 and row 6, where the strokes of | and -
// are; heavy lines are two pixels thick, over columns 2 and 3 and rows 5
// and 6; the strokes of double lines stand a pixel either side of the
// light line, as the bars of = do.
package grid

import (
	"image"
	"math"
	"strings"

	"github.com/firede/platinum-fonts/bitmap"
)

// The cell, in pixels.
const W, H, Ascent = 6, 12, 10

// cell is a cell's pixels, row by row from the top.
type cell [H][W]bool

// fill inks the pixels from x0, y0 to x1, y1, both included.
func (c *cell) fill(x0, y0, x1, y1 int) {
	for y := max(y0, 0); y <= min(y1, H-1); y++ {
		for x := max(x0, 0); x <= min(x1, W-1); x++ {
			c[y][x] = true
		}
	}
}

func (c cell) mirrorX() cell {
	var m cell
	for y := range H {
		for x := range W {
			m[y][x] = c[y][W-1-x]
		}
	}
	return m
}

func (c cell) mirrorY() cell {
	var m cell
	for y := range H {
		m[y] = c[H-1-y]
	}
	return m
}

func (c cell) glyph() bitmap.Glyph {
	return bitmap.NewGlyph(W, image.Rect(0, -Ascent, W, H-Ascent), func(x, y int) bool {
		return c[y+Ascent][x]
	})
}

// Glyphs returns the glyphs grid draws, by character.
func Glyphs() map[rune]bitmap.Glyph {
	m := map[rune]bitmap.Glyph{}
	add := func(r rune, c cell) { m[r] = c.glyph() }
	boxes(add)
	blocks(add)
	braille(add)
	powerline(add)
	return m
}

// The weights of a line.
const (
	none = iota
	light
	heavy
	double
)

// The light lines.
const midX, midY = 2, 6

// cols returns the columns of a vertical line of weight w.
func cols(w int) []int {
	switch w {
	case light:
		return []int{midX}
	case heavy:
		return []int{midX, midX + 1}
	case double:
		return []int{midX - 1, midX + 1}
	}
	return nil
}

// rows returns the rows of a horizontal line of weight w.
func rows(w int) []int {
	switch w {
	case light:
		return []int{midY}
	case heavy:
		return []int{midY - 1, midY}
	case double:
		return []int{midY - 1, midY + 1}
	}
	return nil
}

// boxArms are the box drawing characters from U+2500, each as the weights
// of its arms up, right, down and left; ---- for those drawn otherwise.
const boxArms = `
	0101 0202 1010 2020 ---- ---- ---- ---- // U+2500
	---- ---- ---- ---- 0110 0210 0120 0220 // U+2508
	0011 0012 0021 0022 1100 1200 2100 2200 // U+2510
	1001 1002 2001 2002 1110 1210 2110 1120 // U+2518
	2120 2210 1220 2220 1011 1012 2011 1021 // U+2520
	2021 2012 1022 2022 0111 0112 0211 0212 // U+2528
	0121 0122 0221 0222 1101 1102 1201 1202 // U+2530
	2101 2102 2201 2202 1111 1112 1211 1212 // U+2538
	2111 1121 2121 2112 2211 1122 1221 2212 // U+2540
	1222 2122 2221 2222 ---- ---- ---- ---- // U+2548
	0303 3030 0310 0130 0330 0013 0031 0033 // U+2550
	1300 3100 3300 1003 3001 3003 1310 3130 // U+2558
	3330 1013 3031 3033 0313 0131 0333 1303 // U+2560
	3101 3303 1313 3131 3333 ---- ---- ---- // U+2568
	---- ---- ---- ---- 0001 1000 0100 0010 // U+2570
	0002 2000 0200 0020 0201 1020 0102 2010 // U+2578
`

// arms returns the weights of the arms of the characters in boxArms, by
// character.
func arms() map[rune][4]int {
	m := map[rune][4]int{}
	r := rune(0x2500)
	for _, line := range strings.Split(strings.TrimSpace(boxArms), "\n") {
		line, _, _ = strings.Cut(line, "//")
		for _, a := range strings.Fields(line) {
			if a != "----" {
				m[r] = [4]int{int(a[0] - '0'), int(a[1] - '0'), int(a[2] - '0'), int(a[3] - '0')}
			}
			r++
		}
	}
	return m
}

func boxes(add func(rune, cell)) {
	for r, a := range arms() {
		add(r, box(a[0], a[1], a[2], a[3]))
	}

	// Dashes: double, triple and quadruple, light and heavy. A cell is too
	// narrow for four dashes across; those have three.
	dashes := func(r rune, w int, vertical bool, on func(int) bool) {
		var c cell
		if vertical {
			for _, x := range cols(w) {
				for y := range H {
					c[y][x] = on(y)
				}
			}
		} else {
			for _, y := range rows(w) {
				for x := range W {
					c[y][x] = on(x)
				}
			}
		}
		add(r, c)
	}
	for _, w := range []int{light, heavy} {
		k := rune(w - light)
		dashes(0x2504+k, w, false, func(x int) bool { return x%2 == 0 })
		dashes(0x2506+k, w, true, func(y int) bool { return y%4 < 2 })
		dashes(0x2508+k, w, false, func(x int) bool { return x%2 == 0 })
		dashes(0x250A+k, w, true, func(y int) bool { return y%3 < 2 })
		dashes(0x254C+k, w, false, func(x int) bool { return x%3 < 2 })
		dashes(0x254E+k, w, true, func(y int) bool { return y%6 < 4 })
	}

	// Arcs: the corner's pixel left out, its arms meeting at a slant.
	arc := func(r rune, up, right bool) {
		var c cell
		if up {
			c.fill(midX, 0, midX, midY-1)
		} else {
			c.fill(midX, midY+1, midX, H-1)
		}
		if right {
			c.fill(midX+1, midY, W-1, midY)
		} else {
			c.fill(0, midY, midX-1, midY)
		}
		add(r, c)
	}
	arc(0x256D, false, true)
	arc(0x256E, false, false)
	arc(0x256F, true, false)
	arc(0x2570, true, true)

	// Diagonals, corner to corner, two rows to a column.
	var rising, falling cell
	for y := range H {
		rising[y][(H-1-y)/2] = true
		falling[y][y/2] = true
	}
	add(0x2571, rising)
	add(0x2572, falling)
	var cross cell
	for y := range H {
		for x := range W {
			cross[y][x] = rising[y][x] || falling[y][x]
		}
	}
	add(0x2573, cross)
}

// box draws a box drawing character of arms of weights up, right, down
// and left.
func box(up, right, down, left int) cell {
	var c cell
	if (up == double || down == double) && (left == double || right == double) {
		return doubleBox(up != none, right != none, down != none, left != none)
	}
	// Each arm runs from its edge across the lines it meets, or to the
	// middle of the cell.
	across := append(rows(left), rows(right)...)
	if len(across) == 0 {
		across = []int{midY}
	}
	along := append(cols(up), cols(down)...)
	if len(along) == 0 {
		along = []int{midX}
	}
	top, bottom := minOf(across), maxOf(across)
	first, last := minOf(along), maxOf(along)
	for _, x := range cols(up) {
		c.fill(x, 0, x, bottom)
	}
	for _, x := range cols(down) {
		c.fill(x, top, x, H-1)
	}
	for _, y := range rows(left) {
		c.fill(0, y, last, y)
	}
	for _, y := range rows(right) {
		c.fill(first, y, W-1, y)
	}
	return c
}

// doubleBox draws where double lines meet double lines: each stroke of an
// arm turns into the nearest stroke of the arm on its side, runs on into
// the arm opposite, or turns at a corner into the farther stroke of the
// arm on its other side.
func doubleBox(up, right, down, left bool) cell {
	var c cell
	x0, x1 := midX-1, midX+1 // the strokes of a vertical double line
	y0, y1 := midY-1, midY+1 // and of a horizontal one
	// reach returns where a stroke stops: near if the arm on its side is
	// drawn, mid if the arm opposite is, far otherwise.
	reach := func(side, opposite bool, near, mid, far int) int {
		switch {
		case side:
			return near
		case opposite:
			return mid
		}
		return far
	}
	if up {
		c.fill(x0, 0, x0, reach(left, down, y0, midY, y1))
		c.fill(x1, 0, x1, reach(right, down, y0, midY, y1))
	}
	if down {
		c.fill(x0, reach(left, up, y1, midY, y0), x0, H-1)
		c.fill(x1, reach(right, up, y1, midY, y0), x1, H-1)
	}
	if left {
		c.fill(0, y0, reach(up, right, x0, midX, x1), y0)
		c.fill(0, y1, reach(down, right, x0, midX, x1), y1)
	}
	if right {
		c.fill(reach(up, left, x1, midX, x0), y0, W-1, y0)
		c.fill(reach(down, left, x1, midX, x0), y1, W-1, y1)
	}
	return c
}

func minOf(v []int) int {
	m := v[0]
	for _, x := range v {
		m = min(m, x)
	}
	return m
}

func maxOf(v []int) int {
	m := v[0]
	for _, x := range v {
		m = max(m, x)
	}
	return m
}

// eighths returns k eighths of n pixels, rounded half up.
func eighths(k, n int) int { return (k*n + 4) / 8 }

func blocks(add func(rune, cell)) {
	var c cell
	c.fill(0, 0, W-1, H/2-1)
	add(0x2580, c) // upper half
	for k := 1; k <= 8; k++ {
		var c cell
		c.fill(0, H-eighths(k, H), W-1, H-1)
		add(0x2580+rune(k), c) // lower eighths, to the full block
	}
	for k := 7; k >= 1; k-- {
		var c cell
		c.fill(0, 0, eighths(k, W)-1, H-1)
		add(0x2589+rune(7-k), c) // left eighths
	}
	c = cell{}
	c.fill(W/2, 0, W-1, H-1)
	add(0x2590, c) // right half

	// Shades of a quarter, a half and three quarters, their dots staggered.
	var lightShade, medium, dark cell
	for y := range H {
		for x := range W {
			q := y%2 == 0 && (x+y/2)%2 == 0
			lightShade[y][x], medium[y][x], dark[y][x] = q, (x+y)%2 == 0, !q
		}
	}
	add(0x2591, lightShade)
	add(0x2592, medium)
	add(0x2593, dark)

	c = cell{}
	c.fill(0, 0, W-1, eighths(1, H)-1)
	add(0x2594, c) // upper eighth
	c = cell{}
	c.fill(W-eighths(1, W), 0, W-1, H-1)
	add(0x2595, c) // right eighth

	// Quadrants: upper left, upper right, lower left, lower right.
	const ul, ur, ll, lr = 1, 2, 4, 8
	for i, q := range []int{ll, lr, ul, ul | ll | lr, ul | lr, ul | ur | ll, ul | ur | lr, ur, ur | ll, ur | ll | lr} {
		var c cell
		for k, at := range [][2]int{{0, 0}, {W / 2, 0}, {0, H / 2}, {W / 2, H / 2}} {
			if q&(1<<k) != 0 {
				c.fill(at[0], at[1], at[0]+W/2-1, at[1]+H/2-1)
			}
		}
		add(0x2596+rune(i), c)
	}
}

// braille draws the patterns of eight dots, each two pixels square: in
// two columns and four rows, three pixels apart within a cell and across
// cells.
func braille(add func(rune, cell)) {
	// The column and row of each dot, by its bit.
	dots := [8][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}, {0, 3}, {1, 3}}
	for p := range 256 {
		var c cell
		for bit, d := range dots {
			if p&(1<<bit) != 0 {
				x, y := 3*d[0], 3*d[1]+1
				c.fill(x, y, x+1, y+1)
			}
		}
		add(0x2800+rune(p), c)
	}
}

// powerline draws the separators of Powerline: triangles, half circles and
// slanted halves, solid, and their thin lines.
func powerline(add func(rune, cell)) {
	var triangle, circle, lowerLeft, backslash cell
	r := float64(W) // the half circle's radius
	for y := range H {
		triangle[y] = row(min(y, H-1-y) + 1)
		d := float64(y) + 0.5 - H/2
		circle[y] = row(int(math.Round(math.Sqrt(r*r - d*d))))
		for x := range W {
			lowerLeft[y][x] = y >= 2*x+1
		}
		backslash[y][y/2] = true
	}
	add(0xE0B0, triangle)
	add(0xE0B1, outline(triangle))
	add(0xE0B2, triangle.mirrorX())
	add(0xE0B3, outline(triangle).mirrorX())
	add(0xE0B4, circle)
	add(0xE0B5, outline(circle))
	add(0xE0B6, circle.mirrorX())
	add(0xE0B7, outline(circle).mirrorX())
	add(0xE0B8, lowerLeft)
	add(0xE0B9, backslash)
	add(0xE0BA, lowerLeft.mirrorX())
	add(0xE0BB, backslash.mirrorX())
	add(0xE0BC, lowerLeft.mirrorY())
	add(0xE0BD, backslash.mirrorY())
	add(0xE0BE, lowerLeft.mirrorX().mirrorY())
	add(0xE0BF, backslash.mirrorX().mirrorY())
}

// row returns a row inked from the left for n pixels.
func row(n int) [W]bool {
	var r [W]bool
	for x := range min(n, W) {
		r[x] = true
	}
	return r
}

// outline returns the edge of a shape that stands on the cell's left
// side: its pixels next to paper above, below or to the right of them.
func outline(c cell) cell {
	var o cell
	ink := func(x, y int) bool { return x < 0 || x < W && y >= 0 && y < H && c[y][x] }
	for y := range H {
		for x := range W {
			o[y][x] = c[y][x] && (!ink(x+1, y) || !ink(x, y-1) || !ink(x, y+1) || !ink(x-1, y))
		}
	}
	return o
}
