package ligature

import (
	"image"
	"strings"
	"testing"
)

const sample = `# A comment.

@ ->
............
.......#....
###########.
.......#....

# Another; the rows of a ligature may start with #.
@ ==
............
###########.
............
###########.
`

func TestParse(t *testing.T) {
	ligs, err := Parse("sample", sample, 6, 4, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(ligs) != 2 || ligs[0].Text != "->" || ligs[1].Text != "==" {
		t.Fatalf("%+v", ligs)
	}
	g := ligs[0].Glyph
	if g.Advance != 12 || g.Bounds != image.Rect(0, -3, 12, 1) || !g.At(7, -2) || g.At(11, -1) {
		t.Errorf("-> advances %d in %v", g.Advance, g.Bounds)
	}
	for _, bad := range []string{
		"@ ->\n.....\n", // too narrow, too few rows
		"@ ->\n............\n............\n............\n",
		"@ ->\n............\n............\n............\n..........x.\n",
		"............\n",
	} {
		if _, err := Parse("bad", bad, 6, 4, 3); err == nil {
			t.Errorf("parsed %q", bad)
		}
	}
}

func TestSlice(t *testing.T) {
	ligs, _ := Parse("sample", sample, 6, 4, 3)
	right := Slice(ligs[0].Glyph, 1, 6)
	if right.Advance != 6 || right.Bounds != image.Rect(0, -3, 6, 1) || !right.At(1, -2) || right.At(5, -1) {
		t.Errorf("the right of -> advances %d in %v", right.Advance, right.Bounds)
	}
}

func TestStarts(t *testing.T) {
	texts := []string{"<!--", "===", "-->", "==", "->", "--", "<-"}
	for _, c := range []struct {
		s    string
		want string // the ligature at each character, by its first letter in texts' index, or .
	}{
		{"a == b", "..3..."},
		{"a === b", "..1...."},
		{"====", "...."},
		{"a==", ".3."},
		{"==a==", "3..3."},
		{"x->y->z", ".4..4.."},
		{"--->", "...."},
		{"-->", "2.."},
		{"<!-- x -->", "0......2.."},
		{"<!---", "....."},
		{"<-->", ".2.."},
		{"->->", "4.4."},
	} {
		starts := Starts(texts, []rune(c.s))
		var got strings.Builder
		for _, k := range starts {
			if k < 0 {
				got.WriteByte('.')
			} else {
				got.WriteByte(byte('0' + k))
			}
		}
		if got.String() != c.want {
			t.Errorf("Starts(%q) = %s, want %s", c.s, got.String(), c.want)
		}
	}
}

func TestFeature(t *testing.T) {
	fea := Feature([]string{"==", "->"}, func(r rune) string { return string(r) + "g" },
		func(k, i int) string { return string(rune('a'+k)) + string(rune('0'+i)) })
	for _, want := range []string{
		"    sub a0 =g' by a1;\n",
		"    sub b0 >g' by b1;\n",
		"    ignore sub =g =g' =g;\n",
		"    ignore sub =g' =g =g;\n",
		"    sub =g' =g by a0;\n",
		"    sub -g' >g by b0;\n",
		"feature calt {\n    lookup ligatures;\n} calt;\n",
	} {
		if !strings.Contains(fea, want) {
			t.Errorf("no %q in\n%s", want, fea)
		}
	}
	// Ligatures begun go on before others start.
	if strings.Index(fea, "sub b0") > strings.Index(fea, "ignore") {
		t.Errorf("a ligature begun may be broken by another:\n%s", fea)
	}
}
