package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSignatureHelpAsked(t *testing.T) {
	ct := newEditorTest(t, "go", "fn|")
	var asked []int
	ct.st.OnSignatureHelp = func(r SignatureHelpRequest) {
		asked = append(asked, r.Offset)
	}
	ct.tt.Frame()
	ct.typ("(")
	if len(asked) != 1 || asked[0] != 3 {
		t.Fatalf("asked at %v after (", asked)
	}
	ct.typ(",")
	if len(asked) != 2 || asked[1] != 4 {
		t.Fatalf("asked at %v after ,", asked)
	}
	ct.typ("x")
	if len(asked) != 2 {
		t.Fatalf("asked at %v after x", asked)
	}
}

func TestSignatureHelpCard(t *testing.T) {
	ct := newEditorTest(t, "go", "fn|")
	ct.st.OnSignatureHelp = func(SignatureHelpRequest) {}
	ct.tt.Frame()
	ct.typ("(")
	// The pair closed as it was typed: the caret is in it.
	ct.want("fn(|)")
	ct.st.ShowSignatureHelp(2, 4, "fn(a int)", nil)
	ct.tt.Frame()
	if !ct.tt.HasText("fn(a int)") {
		t.Fatal("the signature does not show")
	}
	// Typing goes on while it shows.
	ct.typ("x")
	ct.want("fn(x|)")
	if !ct.st.ed.sig.open() {
		t.Fatal("the card went while the caret stayed in the call")
	}
	ct.key(0, ui.KeyEscape)
	if ct.st.ed.sig.open() {
		t.Fatal("the card stayed after Escape")
	}
	if !ct.st.Focused() {
		t.Fatal("the keyboard did not come back to the editor")
	}
}

func TestSignatureHelpClosesOutsideTheCall(t *testing.T) {
	ct := newEditorTest(t, "go", "fn|")
	ct.tt.Frame()
	ct.typ("(")
	ct.st.ShowSignatureHelp(2, 4, "fn(a int)", nil)
	ct.tt.Frame()
	if !ct.st.ed.sig.open() {
		t.Fatal("the card did not show")
	}
	ct.st.Select(0, 0)
	ct.tt.Frame()
	if ct.st.ed.sig.open() {
		t.Fatal("the card stayed after the caret left the call")
	}
}
