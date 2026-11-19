package editor

import "testing"

// point returns where the caret at off shows, in the window's DIPs.
func (ct *editorTest) point(off int) (x, y float32) {
	ct.t.Helper()
	ed := &ct.st.ed
	cr, ok := ed.caretRect(off)
	if !ok {
		ct.t.Fatalf("offset %d doesn't show", off)
	}
	return ed.textX + cr.X - ed.scrollX + 1, float32(float64(cr.Y)-ed.scrollY) + 2
}

// drag drags from one offset of the text to another.
func (ct *editorTest) drag(from, to int) {
	ct.t.Helper()
	x, y := ct.point(from)
	ct.tt.Press(x, y)
	x, y = ct.point(to)
	ct.tt.Move(x, y)
	ct.tt.Release(x, y)
}

func TestDragMovesTheSelection(t *testing.T) {
	ct := newEditorTest(t, "", "abcdef")
	ct.st.Select(0, 3)
	ct.drag(2, 6)
	if got := ct.st.Buffer.String(); got != "defabc" {
		t.Fatalf("moved to %q", got)
	}
	// The text moved keeps its selection where it landed.
	ct.want("def[abc]")
	// One undo takes the move back.
	ct.run(Undo)
	ct.want("[abc]def")
}

func TestDragMovesBeforeWhereItWas(t *testing.T) {
	ct := newEditorTest(t, "", "abcdef")
	ct.st.Select(3, 6)
	ct.drag(4, 0)
	if got := ct.st.Buffer.String(); got != "defabc" {
		t.Fatalf("moved to %q", got)
	}
	ct.want("[def]abc")
}

func TestDragMovesAcrossLines(t *testing.T) {
	ct := newEditorTest(t, "", "one\ntwo\n")
	ct.st.Select(0, 3)
	ct.drag(1, 8)
	if got := ct.st.Buffer.String(); got != "\ntwo\none" {
		t.Fatalf("moved to %q", got)
	}
	ct.want("\ntwo\n[one]")
}

func TestDragInsideTheSelectionIsAClick(t *testing.T) {
	ct := newEditorTest(t, "", "abcdef")
	ct.st.Select(0, 3)
	ct.drag(1, 2)
	if got := ct.st.Buffer.String(); got != "abcdef" {
		t.Fatalf("moved to %q", got)
	}
	ct.want("ab|cdef")
}

func TestDragMovesNothingReadOnly(t *testing.T) {
	ct := newEditorTest(t, "", "abcdef")
	ct.st.ReadOnly = true
	ct.st.Select(0, 3)
	ct.drag(2, 6)
	if got := ct.st.Buffer.String(); got != "abcdef" {
		t.Fatalf("moved to %q", got)
	}
}
