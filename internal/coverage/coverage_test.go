package coverage

import "testing"

// TestSetSizes checks the sets against the sizes their standards give.
func TestSetSizes(t *testing.T) {
	want := map[string]int{
		"GB 2312 level 1 hanzi":    3755,
		"GB 2312 level 2 hanzi":    3008,
		"Big5 frequent hanzi":      5401,
		"Big5 less frequent hanzi": 7652,
		"JIS X 0208 kana":          169,
		"JIS X 0208 level 1 kanji": 2965,
		"JIS X 0208 level 2 kanji": 3390,
		"KS X 1001 hangul":         2350,
		"KS X 1001 hanja":          4888,
	}
	for _, s := range Sets() {
		if len(s.Runes) != want[s.Name] {
			t.Errorf("%s: %d characters, want %d", s.Name, len(s.Runes), want[s.Name])
		}
		seen := map[rune]bool{}
		for _, r := range s.Runes {
			if seen[r] {
				t.Errorf("%s: %U twice", s.Name, r)
			}
			seen[r] = true
		}
	}
}
