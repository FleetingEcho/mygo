package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// A drag from the blank block below the text keeps following the finger
// however far the text is above it: a long file drags hundreds of DIPs
// a pixel, and the block must stay built through its own gesture.
func TestMinimapDragFromBlankFollowsTheFinger(t *testing.T) {
	state := &State{Buffer: NewBuffer(strings.Repeat("value\n", 3000))}
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Fill().Children(func() {
			View(c, state).Grow(1).FillWidth().FillHeight()
			Minimap(c, state).Width(86).FillHeight()
		})
	}, 400, 500)
	tester.Frame()
	state.SetScrollY(state.ed.scrollMax())
	tester.Frame()
	t.Logf("start scroll=%v scrollMax=%v viewH=%v total=%v", state.ScrollY(), state.ed.scrollMax(), state.ed.viewH, state.ed.view.heights.total())
	first, last, blankDIP := minimapBand(&state.ed, 76, state.Buffer.LineCount())
	t.Logf("band rows %d..%d blankDIP=%v", first, last, blankDIP)
	x, y := float32(357), float32(462)
	tester.Press(x, y)
	was := state.ScrollY()
	for step := 1; step <= 4; step++ {
		tester.Move(x, y-float32(3*step))
		tester.Frame()
		now := state.ScrollY()
		t.Logf("step %d: scroll %v -> %v", step, was, now)
		if now >= was {
			t.Fatalf("the drag stopped following the finger at step %d", step)
		}
		was = now
	}
	tester.Release(x, y-12)
}
