// Package coverage counts the characters of the national character sets
// that a face draws.
package coverage

import (
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
)

// Set is a part of a character set: its characters, in Unicode.
type Set struct {
	Name  string
	Runes []rune
}

// Sets returns the parts of GB 2312, Big5, JIS X 0208 and KS X 1001 that
// readers of each language need first: their ideographs by level, their
// kana and their hangul.
func Sets() []Set {
	return []Set{
		{"GB 2312 level 1 hanzi", han(simplifiedchinese.GBK, 0xB0, 0xD7, 0xA1, 0xFE)},
		{"GB 2312 level 2 hanzi", han(simplifiedchinese.GBK, 0xD8, 0xF7, 0xA1, 0xFE)},
		{"Big5 frequent hanzi", big5(0xA440, 0xC67E)},
		{"Big5 less frequent hanzi", big5(0xC940, 0xF9D5)},
		{"JIS X 0208 kana", kana(japanese.EUCJP, 0xA4, 0xA5)},
		{"JIS X 0208 level 1 kanji", han(japanese.EUCJP, 0xB0, 0xCF, 0xA1, 0xFE)},
		{"JIS X 0208 level 2 kanji", han(japanese.EUCJP, 0xD0, 0xF4, 0xA1, 0xFE)},
		{"KS X 1001 hangul", hangul(korean.EUCKR, 0xB0, 0xC8)},
		{"KS X 1001 hanja", han(korean.EUCKR, 0xCA, 0xFD, 0xA1, 0xFE)},
	}
}

// decode returns what the two bytes a, b of e decode to, if one character.
func decode(e encoding.Encoding, a, b int) (rune, bool) {
	out, err := e.NewDecoder().Bytes([]byte{byte(a), byte(b)})
	if err != nil {
		return 0, false
	}
	r, n := utf8.DecodeRune(out)
	if n != len(out) || r == utf8.RuneError {
		return 0, false
	}
	return r, true
}

func collect(e encoding.Encoding, leads, trails [2]int, keep func(rune) bool) []rune {
	var rs []rune
	for a := leads[0]; a <= leads[1]; a++ {
		for b := trails[0]; b <= trails[1]; b++ {
			if r, ok := decode(e, a, b); ok && keep(r) {
				rs = append(rs, r)
			}
		}
	}
	return rs
}

func han(e encoding.Encoding, lead0, lead1, trail0, trail1 int) []rune {
	return collect(e, [2]int{lead0, lead1}, [2]int{trail0, trail1}, func(r rune) bool { return unicode.Is(unicode.Han, r) })
}

func kana(e encoding.Encoding, lead0, lead1 int) []rune {
	return collect(e, [2]int{lead0, lead1}, [2]int{0xA1, 0xFE}, func(r rune) bool {
		return unicode.In(r, unicode.Hiragana, unicode.Katakana)
	})
}

func hangul(e encoding.Encoding, lead0, lead1 int) []rune {
	return collect(e, [2]int{lead0, lead1}, [2]int{0xA1, 0xFE}, func(r rune) bool { return unicode.Is(unicode.Hangul, r) })
}

// big5 returns the ideographs of Big5 from code lo to hi.
func big5(lo, hi int) []rune {
	var rs []rune
	for c := lo; c <= hi; c++ {
		b := c & 0xFF
		if b < 0x40 || b > 0x7E && b < 0xA1 || b == 0xFF {
			continue
		}
		if r, ok := decode(traditionalchinese.Big5, c>>8, b); ok && unicode.Is(unicode.Han, r) {
			rs = append(rs, r)
		}
	}
	return rs
}

// Count returns how many of the set's characters has reports drawn.
func (s Set) Count(has func(rune) bool) int {
	n := 0
	for _, r := range s.Runes {
		if has(r) {
			n++
		}
	}
	return n
}
