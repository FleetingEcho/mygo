package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestBracketDepths(t *testing.T) {
	ct := newEditorTest(t, "go", "func f() {\n\ts := \"})\"\n\t// }\n\tif x {\n\t}\n}\n")
	v := &ct.st.ed.view
	opens, closes := bracketPairsOf(ct.st.ed.language())
	// The brackets of a string or a comment nest no deeper.
	want := []int{0, 1, 1, 1, 2, 1, 0}
	for line, depth := range want {
		if got := v.depths.of(v, line, opens, closes); got != depth {
			t.Fatalf("line %d starts at depth %d, want %d", line, got, depth)
		}
	}
}

func TestBracketDepthsFollowEdits(t *testing.T) {
	ct := newEditorTest(t, "go", "|if x {\n\ty()\n}\n")
	ct.typ("{")
	v := &ct.st.ed.view
	opens, closes := bracketPairsOf(ct.st.ed.language())
	want := []int{0, 2, 2, 1}
	for line, depth := range want {
		if got := v.depths.of(v, line, opens, closes); got != depth {
			t.Fatalf("after the edit, line %d starts at depth %d, want %d", line, got, depth)
		}
	}
}

func TestRainbowBracketColors(t *testing.T) {
	ct := newEditorTest(t, "go", "|f(g(1))")
	ct.st.Theme = &Theme{Brackets: [4]ui.Color{
		ui.Hex("#ff0000"), ui.Hex("#00ff00"), ui.Hex("#0000ff"), ui.Hex("#ffff00"),
	}}
	ct.tt.Frame()
	v := &ct.st.ed.view
	l := v.line(0)
	var cc runColors
	v.colorsOf(l, &cc)
	// Each pair shares its depth's color: f ( g ( 1 ) ).
	for _, w := range []struct{ off, depth int }{
		{1, 0}, {3, 1}, {5, 1}, {6, 0},
	} {
		if got, want := cc.at(l.toDisp(w.off, false), ui.Color{}), v.theme.Brackets[w.depth]; got != want {
			t.Fatalf("the bracket at %d is %v, want the depth %d color %v", w.off, got, w.depth, want)
		}
	}
	// What is not a bracket keeps its token's color.
	if got := cc.at(l.toDisp(0, false), ui.Color{}); got == v.theme.Brackets[0] || got == v.theme.Brackets[1] {
		t.Fatalf("the call's name took a bracket color: %v", got)
	}
}
