// Package ligature reads ligatures drawn in text, and finds where they
// are drawn: by Starts in Go, and by the OpenType feature Feature writes,
// which agree.
//
// A ligature of N characters, each a cell wide, is drawn as N slices, a
// cell each, in place of its characters. Where several ligatures match,
// the first in their order is drawn, so longer ligatures go first. None is
// drawn where its first character follows the same character, or its last
// is followed by the same, so that runs as ==== and .... stay apart.
package ligature

import (
	"bufio"
	"fmt"
	"image"
	"strings"

	"github.com/firede/platinum-fonts/bitmap"
)

// Ligature is a ligature of Text drawn W pixels to a cell: a glyph as
// wide as its cells, advancing as far.
type Ligature struct {
	Text  string
	Glyph bitmap.Glyph
}

// Parse reads ligatures drawn in text, each under a line "@ TEXT", in rows
// of '#' ink and '.' paper, height rows of w columns for each character;
// lines starting with '#' elsewhere are comments. The rows are from the
// top of the cell, ascent of them over the baseline.
func Parse(name, text string, w, height, ascent int) ([]Ligature, error) {
	var ligs []Ligature
	sc := bufio.NewScanner(strings.NewReader(text))
	line := 0
	var cur string
	var rows []string
	end := func() error {
		if cur == "" {
			return nil
		}
		width := w * len([]rune(cur))
		if len(rows) != height {
			return fmt.Errorf("%s:%d: %q has %d rows, not %d", name, line, cur, len(rows), height)
		}
		for _, r := range rows {
			if len(r) != width || strings.Trim(r, "#.") != "" {
				return fmt.Errorf("%s:%d: %q has a row %q, not %d of # and .", name, line, cur, r, width)
			}
		}
		rs := rows
		ligs = append(ligs, Ligature{cur, bitmap.NewGlyph(width, image.Rect(0, -ascent, width, height-ascent), func(x, y int) bool {
			return rs[y+ascent][x] == '#'
		})})
		cur, rows = "", nil
		return nil
	}
	for sc.Scan() {
		line++
		s := strings.TrimRight(sc.Text(), " \t\r")
		switch {
		case strings.HasPrefix(s, "@ "):
			if err := end(); err != nil {
				return nil, err
			}
			cur = strings.TrimSpace(s[2:])
			if len([]rune(cur)) < 2 {
				return nil, fmt.Errorf("%s:%d: a ligature of %q", name, line, cur)
			}
		case cur != "" && len(rows) < height && s != "":
			rows = append(rows, s)
		case s == "" || strings.HasPrefix(s, "#"):
			if err := end(); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("%s:%d: a row outside a ligature", name, line)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if err := end(); err != nil {
		return nil, err
	}
	return ligs, nil
}

// Slice returns the slice of g, a ligature's glyph, in its cell i of w
// pixels: a glyph a cell wide.
func Slice(g bitmap.Glyph, i, w int) bitmap.Glyph {
	box := g.Bounds.Intersect(image.Rect(i*w, g.Bounds.Min.Y, (i+1)*w, g.Bounds.Max.Y))
	return bitmap.NewGlyph(w, box.Sub(image.Pt(i*w, 0)), func(x, y int) bool { return g.At(x+i*w, y) })
}

// Starts returns, for each character of s, the index in texts of the
// ligature drawn from it, or -1.
func Starts(texts []string, s []rune) []int {
	starts := make([]int, len(s))
	for i := range starts {
		starts[i] = -1
	}
	joined := false // whether s[i-1] is drawn in a ligature
	for i := 0; i < len(s); {
		k := match(texts, s, i)
		if k < 0 || !free(texts[k], s, i, joined) {
			i++
			joined = false
			continue
		}
		starts[i] = k
		i += len([]rune(texts[k]))
		joined = true
	}
	return starts
}

// match returns the first of texts that s has at i, or -1.
func match(texts []string, s []rune, i int) int {
	for k, t := range texts {
		rs := []rune(t)
		if i+len(rs) <= len(s) && string(s[i:i+len(rs)]) == t {
			return k
		}
	}
	return -1
}

// free reports whether the ligature of t may be drawn at s[i]: neither its
// first character follows the same, unless that is drawn in a ligature,
// nor its last is followed by the same.
func free(t string, s []rune, i int, joined bool) bool {
	rs := []rune(t)
	if i > 0 && !joined && s[i-1] == rs[0] {
		return false
	}
	j := i + len(rs)
	return j >= len(s) || s[j] != rs[len(rs)-1]
}

// Feature returns an OpenType feature file whose calt feature draws the
// ligatures of texts as Starts finds them: each character becomes its
// slice, named by slice, the characters' glyphs named by glyph.
func Feature(texts []string, glyph func(rune) string, slice func(k, i int) string) string {
	var b strings.Builder
	b.WriteString("languagesystem DFLT dflt;\n\nlookup ligatures {\n")
	// A ligature begun goes on to its end; these come first, so that no
	// other ligature starts within it.
	for k, t := range texts {
		for i, r := range []rune(t)[1:] {
			fmt.Fprintf(&b, "    sub %s %s' by %s;\n", slice(k, i), glyph(r), slice(k, i+1))
		}
	}
	for k, t := range texts {
		rs := []rune(t)
		names := make([]string, len(rs))
		for i, r := range rs {
			names[i] = glyph(r)
		}
		rest := strings.Join(names[1:], " ")
		fmt.Fprintf(&b, "    ignore sub %s %s' %s;\n", names[0], names[0], rest)
		fmt.Fprintf(&b, "    ignore sub %s' %s %s;\n", names[0], rest, names[len(names)-1])
		fmt.Fprintf(&b, "    sub %s' %s by %s;\n", names[0], rest, slice(k, 0))
	}
	b.WriteString("} ligatures;\n\nfeature calt {\n    lookup ligatures;\n} calt;\n")
	return b.String()
}
