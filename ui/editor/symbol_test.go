package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// symbolTest makes an editor of "aaa\nbbb\nccc" with a symbol of each
// line.
func symbolTest(t *testing.T, caret string) *editorTest {
	t.Helper()
	ct := newEditorTest(t, "", caret)
	ct.st.Symbols = []Symbol{
		{Name: "alpha", Start: 0, End: 4},
		{Name: "beta", Start: 4, End: 8},
		{Name: "alphabet", Start: 8, End: 11},
	}
	ct.tt.Frame()
	return ct
}

func TestGoToSymbolMatches(t *testing.T) {
	ct := symbolTest(t, "|aaa\nbbb\nccc")
	ct.run(GoToSymbol)
	sym := &ct.st.ed.symbols
	if !sym.open {
		t.Fatal("the bar closed as it opened")
	}
	if len(sym.matches) != 3 {
		t.Fatalf("the empty text matched %v", sym.matches)
	}
	sym.text = "alph"
	sym.search(&ct.st.ed)
	// The shorter name matches better.
	if len(sym.matches) != 2 || sym.matches[0].Name != "alpha" || sym.matches[1].Name != "alphabet" {
		t.Fatalf(`"alph" matched %v`, sym.matches)
	}
	sym.text = "beta"
	sym.search(&ct.st.ed)
	if len(sym.matches) != 1 || sym.matches[0].Name != "beta" {
		t.Fatalf(`"beta" matched %v`, sym.matches)
	}
	sym.text = "nothing here matches"
	sym.search(&ct.st.ed)
	if len(sym.matches) != 0 {
		t.Fatalf("nonsense matched %v", sym.matches)
	}
}

func TestGoToSymbolJumps(t *testing.T) {
	ct := symbolTest(t, "|aaa\nbbb\nccc")
	ct.key(ui.Cmd|ui.Shift, ui.KeyO)
	if !ct.st.ed.symbols.open {
		t.Fatal("the key did not open the bar")
	}
	ct.typ("beta")
	ct.key(0, ui.KeyEnter)
	if ct.st.ed.symbols.open {
		t.Fatal("the bar stayed open")
	}
	ct.want("aaa\n|bbb\nccc")
}

func TestGoToSymbolArrowsChoose(t *testing.T) {
	ct := symbolTest(t, "|aaa\nbbb\nccc")
	ct.run(GoToSymbol)
	ct.typ("alph")
	sym := &ct.st.ed.symbols
	if len(sym.matches) != 2 {
		t.Fatalf(`"alph" matched %v`, sym.matches)
	}
	second := sym.matches[1]
	ct.key(0, ui.KeyDown)
	ct.key(0, ui.KeyEnter)
	if p := ct.st.CursorPos(); p.Line != 2 {
		t.Fatalf("went to line %d for %q", p.Line, second.Name)
	}
}

func TestGoToSymbolEscapeCloses(t *testing.T) {
	ct := symbolTest(t, "|aaa\nbbb\nccc")
	ct.run(GoToSymbol)
	ct.typ("alph")
	ct.key(0, ui.KeyEscape)
	if ct.st.ed.symbols.open {
		t.Fatal("the bar stayed open")
	}
	ct.want("|aaa\nbbb\nccc")
	if !ct.st.Focused() {
		t.Fatal("the keyboard did not come back to the editor")
	}
}
