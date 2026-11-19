package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestCompletionReplaceAndAdditionalEdits(t *testing.T) {
	ct := newEditorTest(t, "", "foo bar |")
	ct.st.ShowCompletions(-1, []Completion{{
		Label:      "quux",
		Insert:     "quux",
		Replace:    &Range{Start: 4, End: 7},
		Additional: []Edit{{Start: 0, End: 0, Text: "import x\n"}},
	}})
	ct.tt.Frame()
	ct.key(0, ui.KeyTab)
	if got := ct.st.Buffer.String(); got != "import x\nfoo quux " {
		t.Fatalf("completed to %q", got)
	}
	// The additional edit goes with the insertion in one step of undo.
	ct.run(Undo)
	if got := ct.st.Buffer.String(); got != "foo bar " {
		t.Fatalf("undo left %q", got)
	}
}

func TestCompletionAdditionalEditsWithASnippet(t *testing.T) {
	ct := newEditorTest(t, "", "pri|")
	ct.st.ShowCompletions(0, []Completion{{
		Label:      "print",
		Kind:       CompletionSnippet,
		Snippet:    snippetFor(t, "print(${1:value})$0"),
		Additional: []Edit{{Start: 0, End: 0, Text: "import fmt\n"}},
	}})
	ct.tt.Frame()
	ct.key(0, ui.KeyTab)
	if got := ct.st.Buffer.String(); got != "import fmt\nprint(value)" {
		t.Fatalf("completed to %q", got)
	}
	// The fields of the snippet follow the text the extra edit pushed.
	ct.want("import fmt\nprint([value])")
	ct.typ("name")
	ct.key(0, ui.KeyTab)
	ct.want("import fmt\nprint(name)|")
}
