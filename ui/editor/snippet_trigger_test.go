package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSnippetTrigger(t *testing.T) {
	ct := newEditorTest(t, "", "call|")
	ct.st.Snippets = map[string]*Snippet{"call": snippetFor(t, "call(${1:a}, ${2:b})$0")}
	ct.tt.Frame()
	ct.key(0, ui.KeyTab)
	ct.want("call([a], b)")
	ct.typ("x")
	ct.key(0, ui.KeyTab)
	ct.want("call(x, [b])")
	ct.key(0, ui.KeyTab)
	ct.want("call(x, b)|")
	// Each undo takes an edit back: the typing, then the expansion.
	ct.run(Undo)
	ct.want("call([a], b)")
	ct.run(Undo)
	ct.want("call|")
}

func TestSnippetTriggerNeedsAWordStart(t *testing.T) {
	ct := newEditorTest(t, "", "xcall|")
	ct.st.Snippets = map[string]*Snippet{"call": snippetFor(t, "call(${1:a})$0")}
	ct.tt.Frame()
	ct.run(Tab)
	if got := ct.st.Buffer.String(); strings.Contains(got, "(") {
		t.Fatalf("expanded the prefix inside a word: %q", got)
	}
}

func TestSnippetTriggerTakesTheLongestPrefix(t *testing.T) {
	ct := newEditorTest(t, "", "format|")
	ct.st.Snippets = map[string]*Snippet{
		"for":    snippetFor(t, "[for]"),
		"format": snippetFor(t, "[format]"),
	}
	ct.tt.Frame()
	ct.key(0, ui.KeyTab)
	ct.want("[format]|")
}

func TestSnippetTriggerOfSymbols(t *testing.T) {
	// A prefix of symbols expands after a word, as ":d" after "foo".
	ct := newEditorTest(t, "", "foo:d|")
	ct.st.Snippets = map[string]*Snippet{":d": snippetFor(t, "${1:div}")}
	ct.tt.Frame()
	ct.key(0, ui.KeyTab)
	ct.want("foo[div]")
}
