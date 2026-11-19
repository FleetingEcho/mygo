package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

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

func TestStickyBandIsOpaqueAndNumbersItsHeaders(t *testing.T) {
	ct := newEditorTest(t, "go", "func a() {\n\tif x {\n\t\ty()\n\t\tz()\n\t}\n}\n")
	red := ui.RGB(255, 0, 0)
	ct.st.Theme = &Theme{
		Background: ui.RGB(255, 255, 255),
		LineNumber: red, ActiveLineNumber: red,
	}
	ct.st.LineBackgrounds = []LineBackground{{First: 0, Last: 5, Color: ui.RGB(0, 0, 255)}}
	ct.tt.Frame()
	ct.st.SetScrollY(float64(ct.st.ed.view.lineHeight) * 2)
	ct.tt.Frame()
	img := ct.tt.Image()
	s := float32(img.Bounds().Dx()) / 600
	// Under the empty end of each header, no blue line beneath it shows
	// through the band: the band is the editor's opaque background.
	for row := range 2 {
		y := (float32(row)*ct.st.ed.view.lineHeight + 2) * s
		c := img.RGBAAt(int(500*s), int(y))
		if c.R != 255 || c.G != 255 || c.B != 255 {
			t.Fatalf("the empty end of header %d painted %v, want the opaque background", row, c)
		}
	}
	// Both headers number themselves in the gutter, not the line the
	// ordinary scroll geometry would put there.
	redPixels := 0
	for y := range int(2 * ct.st.ed.view.lineHeight * s) {
		for x := range int(ct.st.ed.gutterW * s) {
			c := img.RGBAAt(x, y)
			if c.R > 200 && c.G < 60 && c.B < 60 {
				redPixels++
			}
		}
	}
	if redPixels == 0 {
		t.Fatal("the pinned headers have no line numbers")
	}
}
