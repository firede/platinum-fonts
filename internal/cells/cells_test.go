package cells

import "testing"

func TestOf(t *testing.T) {
	for _, c := range []struct {
		r    rune
		want int
	}{
		{'A', 1}, {'中', 2}, {'，', 2}, {'ｱ', 1}, {'—', 1}, {'│', 1}, {'⣿', 1},
		{'́', 0}, {'​', 0}, {'­', 1}, {'ᅡ', 0}, {'가', 2},
		{'😀', 2}, {'', 1}, {'\U000F0219', 1},
	} {
		if got := Of(c.r); got != c.want {
			t.Errorf("Of(%U) = %d, want %d", c.r, got, c.want)
		}
	}
}

func TestRanges(t *testing.T) {
	rs := Ranges()
	for i, r := range rs {
		if r.Lo > r.Hi || i > 0 && rs[i-1].Hi >= r.Lo || r.Cells == 1 {
			t.Fatalf("range %d: %+v", i, r)
		}
		for _, c := range []rune{r.Lo, r.Hi} {
			if Of(c) != r.Cells {
				t.Fatalf("range %+v: Of(%U) = %d", r, c, Of(c))
			}
		}
	}
	t.Logf("%d ranges", len(rs))
}
