package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Dragging the lit rows at the top of a freshly opened file scrolls
// the view, and the rows touch so the light has no gaps in it.
func TestMinimapTopDragScrolls(t *testing.T) {
	state := &State{Buffer: NewBuffer(strings.Repeat("value\n", 300))}
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Fill().Children(func() {
			View(c, state).Grow(1).FillWidth().FillHeight()
			Minimap(c, state).Width(86).FillHeight()
		})
	}, 400, 500)
	tester.Frame()
	t.Logf("before drag: total=%v viewH=%v scrollMax=%v scroll=%v", state.ed.view.heights.total(), state.ed.viewH, state.ed.scrollMax(), state.ScrollY())
	box, ok := tester.Find("Go to line 1")
	if !ok {
		t.Fatalf("first row not found")
	}
	x, y := box.X+box.W/2, box.Y+box.H/2
	tester.Press(x, y)
	tester.Move(x, y+20)
	tester.Frame()
	t.Logf("after +20: scroll=%v", state.ScrollY())
	tester.Move(x, y+40)
	tester.Frame()
	t.Logf("after +40: scroll=%v", state.ScrollY())
	tester.Release(x, y+40)
	if state.ScrollY() == 0 {
		t.Fatal("dragging the highlight down did not scroll")
	}
}
