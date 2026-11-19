package editor

import "testing"

// TestWordOccurrences lights the other occurrences of the word at a
// caret, as those of a selection.
func TestWordOccurrences(t *testing.T) {
	ct := newEditorTest(t, "", "foo  bar fo|o baz foo")
	ed := &ct.st.ed
	f := &frame{first: 0, last: 1}
	ed.prepare(f)
	if f.occurrence != "foo" {
		t.Fatalf("the word at the caret lit %q", f.occurrence)
	}
	// A selection lights its own text.
	ct.st.Select(5, 8)
	f = &frame{first: 0, last: 1}
	ed.prepare(f)
	if f.occurrence != "bar" {
		t.Fatalf("the selection lit %q", f.occurrence)
	}
	// A caret at a word's end, before the space, lights the word still.
	ct.st.Select(3, 3)
	f = &frame{first: 0, last: 1}
	ed.prepare(f)
	if f.occurrence != "foo" {
		t.Fatalf("the word's end lit %q", f.occurrence)
	}
	// Whitespace at a caret lights nothing.
	ct.st.Select(4, 4)
	f = &frame{first: 0, last: 1}
	ed.prepare(f)
	if f.occurrence != "" {
		t.Fatalf("the whitespace at the caret lit %q", f.occurrence)
	}
}
