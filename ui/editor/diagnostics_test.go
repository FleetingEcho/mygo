package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// diagTest makes an editor of "aaa\nbbb\nccc", as marked with carets,
// with one diagnostic at the start of each of the last two lines.
func diagTest(t *testing.T, caret string) *editorTest {
	t.Helper()
	ct := newEditorTest(t, "", caret)
	ct.st.SetDiagnostics(ct.st.Buffer.Version(), []Diagnostic{
		{Start: 4, End: 7, Message: "b"},
		{Start: 8, End: 11, Message: "c"},
	})
	return ct
}

func TestNextDiagnostic(t *testing.T) {
	ct := diagTest(t, "|aaa\nbbb\nccc")
	ct.run(NextDiagnostic)
	ct.want("aaa\n|bbb\nccc")
	ct.run(NextDiagnostic)
	ct.want("aaa\nbbb\n|ccc")
	// Wraps around the end of the text.
	ct.run(NextDiagnostic)
	ct.want("aaa\n|bbb\nccc")
}

func TestPreviousDiagnostic(t *testing.T) {
	ct := diagTest(t, "aaa\nbbb\n|ccc")
	ct.run(PreviousDiagnostic)
	ct.want("aaa\n|bbb\nccc")
	// Wraps around the start of the text.
	ct.run(PreviousDiagnostic)
	ct.want("aaa\nbbb\n|ccc")
}

func TestDiagnosticNavigationSkipsTheDiagnosticAtTheCaret(t *testing.T) {
	ct := diagTest(t, "aaa\n|bbb\nccc")
	ct.run(NextDiagnostic)
	ct.want("aaa\nbbb\n|ccc")
	ct = diagTest(t, "aaa\nbbb\n|ccc")
	ct.run(PreviousDiagnostic)
	ct.want("aaa\n|bbb\nccc")
}

func TestDiagnosticNavigationOfNone(t *testing.T) {
	ct := newEditorTest(t, "", "|aaa\nbbb")
	if ct.st.Run(NextDiagnostic) || ct.st.Run(PreviousDiagnostic) {
		t.Fatal("navigated without diagnostics")
	}
	ct.want("|aaa\nbbb")
}

func TestDiagnosticNavigationKey(t *testing.T) {
	ct := diagTest(t, "|aaa\nbbb\nccc")
	ct.key(0, ui.KeyF8)
	ct.want("aaa\n|bbb\nccc")
	ct.key(0, ui.KeyF8)
	ct.want("aaa\nbbb\n|ccc")
	ct.key(ui.Shift, ui.KeyF8)
	ct.want("aaa\n|bbb\nccc")
}
