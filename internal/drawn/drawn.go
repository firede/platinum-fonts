// Package drawn reads faces drawn in text: each glyph's rows under a line
// naming its character, '#' ink and any other character paper.
//
//	@height 12
//	@ascent 9
//	== A
//	..#..
//	.#.#.
//
// The directives give the cell's height and the rows of it over the
// baseline (ascent), the design's cap height and x-height, and the blank
// columns after every glyph (tracking, 1 unless given). A glyph line is
// "== X", X the character, "U+XXXX" its code point in hexadecimal and
// optionally a name after it, or "space N" for a blank glyph N columns
// wide. Blank lines are ignored;
// rows missing at the bottom of a glyph are blank, and a glyph is as wide
// as its longest row.
//
// Combining marks (U+0300–U+036F) are not glyphs: the accented letters are
// composed of a letter and a mark. A mark goes a row clear of a small
// letter's top (an i or j losing its dot), on a capital's, rising over the
// cell by up to MaxAbove rows, or under the baseline for a cedilla, comma
// or ogonek, the ogonek under the letter's right side.
package drawn

import (
	"bufio"
	"fmt"
	"image"
	"strconv"
	"strings"

	"github.com/firede/platinum-fonts/bitmap"
)

// MaxAbove is how many rows a mark may rise over the cell.
const MaxAbove = 2

// Face is a parsed face.
type Face struct {
	Height, Ascent, CapHeight, XHeight, Tracking int
	Glyphs                                       map[rune]bitmap.Glyph
}

// Parse reads a face; name is for its errors.
func Parse(name, src string) (*Face, error) {
	f := &Face{Tracking: 1, Glyphs: map[rune]bitmap.Glyph{}}
	rows := map[rune][]string{}
	widths := map[rune]int{}
	var order []rune
	cur := rune(-1)
	sc := bufio.NewScanner(strings.NewReader(src))
	line := 0
	for sc.Scan() {
		line++
		s := strings.TrimRight(sc.Text(), " \t\r")
		switch {
		case s == "":
		case strings.HasPrefix(s, "@"):
			fields := strings.Fields(s[1:])
			if len(fields) != 2 {
				return nil, fmt.Errorf("%s:%d: bad directive %q", name, line, s)
			}
			n, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %v", name, line, err)
			}
			switch fields[0] {
			case "height":
				f.Height = n
			case "ascent":
				f.Ascent = n
			case "capheight":
				f.CapHeight = n
			case "xheight":
				f.XHeight = n
			case "tracking":
				f.Tracking = n
			default:
				return nil, fmt.Errorf("%s:%d: unknown directive %q", name, line, fields[0])
			}
		case strings.HasPrefix(s, "== "):
			spec := s[3:]
			var r rune
			switch {
			case strings.HasPrefix(spec, "space "):
				w, err := strconv.Atoi(strings.TrimSpace(spec[6:]))
				if err != nil {
					return nil, fmt.Errorf("%s:%d: %v", name, line, err)
				}
				r = ' '
				widths[r] = w
			case strings.HasPrefix(spec, "U+"):
				code, _, _ := strings.Cut(spec[2:], " ")
				n, err := strconv.ParseInt(code, 16, 32)
				if err != nil {
					return nil, fmt.Errorf("%s:%d: %v", name, line, err)
				}
				r = rune(n)
			default:
				rs := []rune(spec)
				if len(rs) != 1 {
					return nil, fmt.Errorf("%s:%d: bad glyph name %q", name, line, spec)
				}
				r = rs[0]
			}
			if _, dup := rows[r]; dup {
				return nil, fmt.Errorf("%s:%d: glyph %q defined twice", name, line, r)
			}
			rows[r] = nil
			order = append(order, r)
			cur = r
		default:
			if cur < 0 {
				return nil, fmt.Errorf("%s:%d: row outside a glyph", name, line)
			}
			rows[cur] = append(rows[cur], s)
		}
	}
	if f.Height <= 0 || f.Ascent <= 0 {
		return nil, fmt.Errorf("%s: missing @height or @ascent", name)
	}
	marks := map[rune][]string{}
	kept := order[:0]
	for _, r := range order {
		if isMark(r) {
			marks[r] = rows[r]
			delete(rows, r)
		} else {
			kept = append(kept, r)
		}
	}
	order = kept
	over := map[rune][]string{} // rows over the cell, the last nearest it
	for _, c := range composites {
		r, base, mark := c[0], c[1], c[2]
		if _, has := rows[r]; has || rows[base] == nil || marks[mark] == nil {
			continue
		}
		below := mark == 0x327 || mark == 0x328 || mark == 0x326
		rows[r], over[r] = f.compose(rows[base], marks[mark], below, mark == 0x328, base == 'i' || base == 'j')
		order = append(order, r)
	}
	for _, r := range order {
		rs := rows[r]
		if len(rs) > f.Height {
			return nil, fmt.Errorf("%s: glyph %q is %d rows tall, more than %d", name, r, len(rs), f.Height)
		}
		w := widths[r]
		for _, row := range append(rs, over[r]...) {
			w = max(w, len([]rune(row)))
		}
		up := over[r]
		bounds := image.Rect(0, -f.Ascent-len(up), w, f.Height-f.Ascent)
		f.Glyphs[r] = bitmap.NewGlyph(w+f.Tracking, bounds, func(x, y int) bool {
			var row string
			if i := y + f.Ascent; i >= 0 {
				if i >= len(rs) {
					return false
				}
				row = rs[i]
			} else {
				row = up[len(up)+i]
			}
			cells := []rune(row)
			return x < len(cells) && cells[x] == '#'
		})
	}
	return f, nil
}

func isMark(r rune) bool { return r >= 0x300 && r <= 0x36F }

// composites are the letters made of a letter and a mark: the letter, its
// base and its mark.
var composites = func() [][3]rune {
	var out [][3]rune
	for _, set := range []struct {
		mark        rune
		bases, made string
	}{
		{0x300, "AEIOUaeiou", "ÀÈÌÒÙàèìòù"},
		{0x301, "AEIOUYaeiouy", "ÁÉÍÓÚÝáéíóúý"},
		{0x302, "AEIOUaeiou", "ÂÊÎÔÛâêîôû"},
		{0x303, "ANOano", "ÃÑÕãñõ"},
		{0x308, "AEIOUaeiouy", "ÄËÏÖÜäëïöüÿ"},
		{0x30A, "Aa", "Åå"},
		{0x327, "CcSs", "ÇçŞş"},
		// Latin Extended-A: Central European and Turkish.
		{0x301, "CNSZLRcnszlr", "ĆŃŚŹĹŔćńśźĺŕ"},
		{0x30C, "CENRSZTcenrsz", "ČĚŇŘŠŽŤčěňřšž"},
		{0x30C, "D", "Ď"},
		{0x307, "ZEIze", "ŻĖİżė"},
		{0x30B, "OUou", "ŐŰőű"},
		{0x306, "GAga", "ĞĂğă"},
		{0x30A, "Uu", "Ůů"},
		{0x328, "AEae", "ĄĘąę"},
		// Romanian and Latvian.
		{0x304, "AEIUaeiu", "ĀĒĪŪāēīū"},
		{0x326, "SsTtKkLlNn", "ȘșȚțĶķĻļŅņ"},
	} {
		made := []rune(set.made)
		for i, b := range []rune(set.bases) {
			out = append(out, [3]rune{made[i], b, set.mark})
		}
	}
	return out
}()

// compose draws base with mark over it, or under it if below (under its
// right side if right), centered; dotless takes the dot off an i or j
// first. It returns the rows of the cell, and those over it.
func (f *Face) compose(base, mark []string, below, right, dotless bool) (rows, above []string) {
	bw, mw := 0, 0
	for _, r := range base {
		bw = max(bw, len([]rune(r)))
	}
	for _, r := range mark {
		mw = max(mw, len([]rune(r)))
	}
	w := max(bw, mw)
	cells := make([][]rune, f.Height)
	for y := range cells {
		cells[y] = []rune(strings.Repeat(".", w))
	}
	put := func(y, x int, row string, set [][]rune) {
		for i, ch := range []rune(row) {
			if ch == '#' && y >= 0 && y < len(set) && x+i < w {
				set[y][x+i] = '#'
			}
		}
	}
	top := f.Height
	for y, r := range base {
		if strings.Contains(r, "#") && !(dotless && y < f.Ascent-f.XHeight) {
			top = min(top, y)
		}
	}
	for y, r := range base {
		if dotless && y < f.Ascent-f.XHeight {
			continue
		}
		put(y, (w-bw)/2, r, cells)
	}
	if below {
		x := (w - mw) / 2
		if right {
			x = max((w-bw)/2+bw-mw, 0)
		}
		for i, r := range mark {
			put(f.Ascent+i, x, r, cells)
		}
	} else {
		// A row clear of the letter's top where the leading has room; on a
		// capital, rising over the cell.
		first := max(top-1-len(mark), -MaxAbove)
		var over [][]rune
		if first < 0 {
			over = make([][]rune, -first)
			for y := range over {
				over[y] = []rune(strings.Repeat(".", w))
			}
		}
		for i, r := range mark {
			if y := first + i; y < 0 {
				put(y+len(over), (w-mw)/2, r, over)
			} else {
				put(y, (w-mw)/2, r, cells)
			}
		}
		for _, r := range over {
			above = append(above, string(r))
		}
	}
	for _, r := range cells {
		rows = append(rows, string(r))
	}
	return rows, above
}
