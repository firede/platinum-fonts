// Package cells tells how many cells of a terminal's grid a character
// takes, as wcwidth computes it: two for East Asian Wide and Fullwidth
// characters, none for marks, format characters, controls and the
// medial vowels and final consonants of Hangul, and one for every other,
// East Asian Ambiguous among them.
package cells

import (
	"unicode"

	"golang.org/x/text/width"
)

// UnicodeVersion is the version of Unicode Of follows.
const UnicodeVersion = width.UnicodeVersion

// Of returns the cells r takes.
func Of(r rune) int {
	switch {
	case r == 0xAD: // a soft hyphen is shown where a line breaks at it
		return 1
	case r >= 0x1160 && r <= 0x11FF, r >= 0xD7B0 && r <= 0xD7FF:
		return 0
	case unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf, unicode.Cc):
		return 0
	}
	switch width.LookupRune(r).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	}
	return 1
}

// Range is a run of characters taking the same cells.
type Range struct {
	Lo, Hi rune
	Cells  int
}

// Ranges returns the runs of characters that do not take one cell, in
// order.
func Ranges() []Range {
	var rs []Range
	for r := rune(0); r <= unicode.MaxRune; r++ {
		n := Of(r)
		if n == 1 {
			continue
		}
		if k := len(rs) - 1; k >= 0 && rs[k].Hi == r-1 && rs[k].Cells == n {
			rs[k].Hi = r
		} else {
			rs = append(rs, Range{r, r, n})
		}
	}
	return rs
}
