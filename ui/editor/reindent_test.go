package editor

import "testing"

func TestReindentLines(t *testing.T) {
	// The middle line is indented too much and the closing one too
	// little: both go where the language puts them.
	ct := newEditorTest(t, "go", "func f() {\n        x()\n}\n")
	ct.run(SelectAll)
	ct.run(Reindent)
	if got := ct.st.Buffer.String(); got != "func f() {\n\tx()\n}\n" {
		t.Fatalf("reindented to %q", got)
	}
	// A line closing a block and opening one again, as Go's else.
	ct = newEditorTest(t, "go", "if a {\nb()\n  } else {\nc()\n}\n")
	ct.run(SelectAll)
	ct.run(Reindent)
	if got := ct.st.Buffer.String(); got != "if a {\n\tb()\n} else {\n\tc()\n}\n" {
		t.Fatalf("reindented else to %q", got)
	}
	// Blank lines keep as they are.
	ct = newEditorTest(t, "go", "func f() {\n\n   \n  x()\n}\n")
	ct.run(SelectAll)
	ct.run(Reindent)
	if got := ct.st.Buffer.String(); got != "func f() {\n\n   \n\tx()\n}\n" {
		t.Fatalf("reindented through blanks to %q", got)
	}
}

func TestReindentLinesOfEachSelection(t *testing.T) {
	ct := newEditorTest(t, "go", "if a {\nx()\n}\nif b {\ny()\n}\n")
	ct.st.SetSelections(Selection{Anchor: 8, Head: 11}, Selection{Anchor: 18, Head: 21})
	ct.run(Reindent)
	if got := ct.st.Buffer.String(); got != "if a {\n\tx()\n}\nif b {\n\ty()\n}\n" {
		t.Fatalf("reindented to %q", got)
	}
}

func TestReindentLinesNeedsRules(t *testing.T) {
	// Without a language nothing says how the lines indent: they keep
	// as they are.
	ct := newEditorTest(t, "", "a\n  b\n")
	if ct.st.Run(Reindent) {
		t.Fatal("reindented without a language")
	}
	if got := ct.st.Buffer.String(); got != "a\n  b\n" {
		t.Fatalf("reindented to %q", got)
	}
}
