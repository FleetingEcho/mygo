package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestApplyEdits(t *testing.T) {
	ct := newEditorTest(t, "", "one two three")
	if !ct.st.ApplyEdits([]Edit{{Start: 4, End: 7, Text: "2"}, {Start: 0, End: 3, Text: "1"}}) {
		t.Fatal("the edits did not apply")
	}
	if got := ct.st.Buffer.String(); got != "1 2 three" {
		t.Fatalf("applied to %q", got)
	}
	// Together, in one step of undo.
	ct.run(Undo)
	if got := ct.st.Buffer.String(); got != "one two three" {
		t.Fatalf("undo left %q", got)
	}
}

func TestRename(t *testing.T) {
	ct := newEditorTest(t, "", "foo bar foo|")
	var asked []RenameRequest
	ct.st.OnRename = func(r RenameRequest) { asked = append(asked, r) }
	ct.tt.Frame()
	ct.run(Rename)
	if len(asked) != 1 || asked[0].Offset != 11 {
		t.Fatalf("asked %+v", asked)
	}
	ct.st.ShowRename(8, 11, "foo")
	ct.tt.Frame()
	if !ct.st.ed.rename.open {
		t.Fatal("the bar did not open")
	}
	ct.typ("baz")
	ct.key(0, ui.KeyEnter)
	if ct.st.ed.rename.open {
		t.Fatal("the bar stayed open")
	}
	if len(asked) != 2 {
		t.Fatalf("asked %+v", asked)
	}
	r := asked[1]
	if r.NewName != "baz" || r.OldName != "foo" || r.Start != 8 || r.End != 11 {
		t.Fatalf("asked %+v", r)
	}
	// The app makes the edits it asked for.
	ct.st.ApplyEdits([]Edit{{Start: 8, End: 11, Text: r.NewName}})
	if got := ct.st.Buffer.String(); got != "foo bar baz" {
		t.Fatalf("renamed to %q", got)
	}
}

func TestRenameWithoutAnApp(t *testing.T) {
	// Without an app to ask, every occurrence of the word is selected,
	// which typing renames at once.
	ct := newEditorTest(t, "", "foo bar fo|o")
	ct.run(Rename)
	ct.typ("baz")
	if got := ct.st.Buffer.String(); got != "baz bar baz" {
		t.Fatalf("renamed to %q", got)
	}
}

func TestCodeActions(t *testing.T) {
	ct := newEditorTest(t, "", "hello| world")
	var asked []CodeActionRequest
	ct.st.OnCodeActions = func(r CodeActionRequest) { asked = append(asked, r) }
	ct.tt.Frame()
	ct.run(CodeActions)
	if len(asked) != 1 || asked[0].Offset != 5 {
		t.Fatalf("asked %+v", asked)
	}
	ct.st.ShowCodeActions([]CodeAction{
		{Title: "Uppercase it", Edits: []Edit{{Start: 0, End: 5, Text: "HELLO"}}},
		{Title: "Delete it", Edits: []Edit{{Start: 0, End: 5}}},
	})
	ct.tt.Frame()
	if !ct.tt.HasText("Uppercase it") || !ct.tt.HasText("Delete it") {
		t.Fatal("the menu does not show")
	}
	ct.key(0, ui.KeyDown)
	ct.key(0, ui.KeyEnter)
	if got := ct.st.Buffer.String(); got != " world" {
		t.Fatalf("the action made %q", got)
	}
	if ct.st.ed.actions.shown {
		t.Fatal("the menu stayed open")
	}
}
