package editor

import "testing"

// overInlay finds where the inlay holding text shows, in the window's
// DIPs.
func (ct *editorTest) overInlay(text string) (x, y float32, ok bool) {
	ct.t.Helper()
	ed := &ct.st.ed
	for y := float32(0); y < 400; y += 2 {
		for x := float32(0); x < 600; x += 2 {
			cx, cy := ed.toContent(x, y)
			if in, hit := ed.inlayAt(cx, float64(cy)); hit && in.text == text {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

func TestInlayClick(t *testing.T) {
	ct := newEditorTest(t, "", "hello\nworld")
	clicked := 0
	ct.st.SetInlays([]Inlay{
		{Offset: 5, Text: " 2024", Trailing: true, Click: func() { clicked++ }},
		{Offset: 11, Text: " none", Trailing: true},
	})
	ct.tt.Frame()
	x, y, ok := ct.overInlay(" 2024")
	if !ok {
		t.Fatal("the inlay does not show")
	}
	ct.tt.Press(x, y)
	ct.tt.Release(x, y)
	if clicked != 1 {
		t.Fatalf("the click ran %d times", clicked)
	}
	// The click took the press from the text: the caret did not move.
	if got := ct.st.Cursor(); got != 0 {
		t.Fatalf("the caret went to %d", got)
	}
}

func TestInlayClickWithoutOne(t *testing.T) {
	// An inlay of no click takes no press from the text.
	ct := newEditorTest(t, "", "hello\nworld")
	ct.st.SetInlays([]Inlay{{Offset: 11, Text: " none", Trailing: true}})
	ct.tt.Frame()
	x, y, ok := ct.overInlay(" none")
	if !ok {
		t.Fatal("the inlay does not show")
	}
	ct.tt.Press(x, y)
	ct.tt.Release(x, y)
	if p := ct.st.CursorPos(); p.Line != 1 {
		t.Fatalf("the caret went to line %d", p.Line)
	}
}
