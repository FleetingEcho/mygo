package editor

import "testing"

func TestScopeHeaders(t *testing.T) {
	ct := newEditorTest(t, "go", "func f() {\n\tif x {\n\t\ty()\n\t}\n}\n")
	ed := &ct.st.ed
	want := map[int][]int{
		0: nil,
		1: {0},
		2: {1, 0},
		3: {1, 0},
		4: {0},
		5: nil,
	}
	for line, w := range want {
		got := ed.scopeHeaders(line)
		if len(got) != len(w) {
			t.Fatalf("the scopes of line %d are %v, want %v", line, got, w)
		}
		for i := range w {
			if got[i] != w[i] {
				t.Fatalf("the scopes of line %d are %v, want %v", line, got, w)
			}
		}
	}
}

func TestStickyHeightFollowsTheScroll(t *testing.T) {
	ct := newEditorTest(t, "go", "func f() {\n\tx()\n\ty()\n}\n")
	ed := &ct.st.ed
	// At the top of the text the function's line shows where it is, and
	// sticks to nothing.
	if got := ed.stickyHeight(); got != 0 {
		t.Fatalf("the top of the text takes %v of sticky headers", got)
	}
	// Scrolled into the body, the function's line sticks.
	ct.st.SetScrollY(float64(ed.view.lineHeight) * 1.5)
	ct.tt.Frame()
	if got, want := ed.stickyHeight(), ed.view.lineHeight; got != want {
		t.Fatalf("the text scrolled into its body takes %v of sticky headers, want %v", got, want)
	}
}

// A press on the pinned headers goes to their line, as VS Code's sticky
// scroll: the line scrolled to the top of the text.
func TestStickyHeaderClickGoesToItsLine(t *testing.T) {
	ct := newEditorTest(t, "go", "func f() {\n\tif x {\n\t\ty()\n\t\tz()\n\t}\n}\n")
	ed := &ct.st.ed
	ct.st.SetScrollY(float64(ed.view.lineHeight) * 2)
	ct.tt.Frame()
	if ed.sticky <= 0 {
		t.Fatal("no scope headers pinned")
	}
	// The band holds the function's line above the if's: pressing the
	// lower of the two goes to the if's line.
	ct.tt.Press(20, float32(ed.view.lineHeight)*1.5)
	ct.tt.Release(20, float32(ed.view.lineHeight)*1.5)
	if got, want := ct.st.ScrollY(), float64(ed.view.lineHeight); got != want {
		t.Fatalf("the press scrolled to %v, want the if's line at %v", got, want)
	}
	// The function's line alone pins again above it.
	if got := ed.stickyHeight(); got != float32(ed.view.lineHeight) {
		t.Fatalf("the line scrolled to takes %v of sticky headers", got)
	}
}
