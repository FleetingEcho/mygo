package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestCommandPaletteMatches(t *testing.T) {
	if len(commandNames) != int(commandCount) {
		t.Fatalf("%d names for %d commands", len(commandNames), commandCount)
	}
	ct := newEditorTest(t, "", "|")
	ct.run(CommandPalette)
	pal := &ct.st.ed.palette
	if !pal.open {
		t.Fatal("the palette closed as it opened")
	}
	// Every command but NoCommand, in their order, while the text is
	// empty.
	if len(pal.matches) != int(commandCount)-1 {
		t.Fatalf("the empty text matched %d of %d commands", len(pal.matches), int(commandCount)-1)
	}
	for i, c := range pal.matches {
		if c != Command(i+1) {
			t.Fatalf("command %d of the empty text is %v", i, c)
		}
	}
	// Typed text matches the commands holding its letters in order.
	pal.text = "diag"
	pal.search()
	if len(pal.matches) != 2 || pal.matches[0] != NextDiagnostic || pal.matches[1] != PreviousDiagnostic {
		t.Fatalf(`"diag" matched %v`, pal.matches)
	}
	pal.text = "uppercase"
	pal.search()
	if len(pal.matches) != 1 || pal.matches[0] != Uppercase {
		t.Fatalf(`"uppercase" matched %v`, pal.matches)
	}
	pal.text = "nothing here matches"
	pal.search()
	if len(pal.matches) != 0 {
		t.Fatalf("nonsense matched %v", pal.matches)
	}
}

func TestCommandPaletteStepWraps(t *testing.T) {
	ct := newEditorTest(t, "", "|")
	ct.run(CommandPalette)
	pal := &ct.st.ed.palette
	pal.text = "diag"
	pal.search()
	pal.step(1)
	if pal.chosen != 1 {
		t.Fatalf("the choice is %d after down", pal.chosen)
	}
	pal.step(1)
	if pal.chosen != 0 {
		t.Fatalf("the choice is %d after down at the end", pal.chosen)
	}
	pal.step(-1)
	if pal.chosen != 1 {
		t.Fatalf("the choice is %d after up at the start", pal.chosen)
	}
}

func TestCommandPaletteRunsACommand(t *testing.T) {
	ct := newEditorTest(t, "", "[aaa] bbb")
	ct.key(ui.Cmd|ui.Shift, ui.KeyP)
	if !ct.st.ed.palette.open {
		t.Fatal("the key did not open the palette")
	}
	ct.typ("uppercase")
	ct.key(0, ui.KeyEnter)
	if ct.st.ed.palette.open {
		t.Fatal("the palette stayed open after running a command")
	}
	ct.want("[AAA] bbb")
	if !ct.st.Focused() {
		t.Fatal("the keyboard did not come back to the editor")
	}
}

func TestCommandPaletteArrowsChoose(t *testing.T) {
	ct := diagTest(t, "|aaa\nbbb\nccc")
	ct.run(CommandPalette)
	ct.typ("diagnostic")
	pal := &ct.st.ed.palette
	if len(pal.matches) != 2 {
		t.Fatalf(`"diagnostic" matched %v`, pal.matches)
	}
	second := pal.matches[1]
	ct.key(0, ui.KeyDown)
	ct.key(0, ui.KeyEnter)
	switch second {
	case NextDiagnostic:
		ct.want("aaa\n|bbb\nccc")
	case PreviousDiagnostic:
		// The one before the caret wraps to the last.
		ct.want("aaa\nbbb\n|ccc")
	default:
		t.Fatalf("the choice ran %v", second)
	}
}

func TestCommandPaletteEscapeCloses(t *testing.T) {
	ct := newEditorTest(t, "", "|aaa")
	ct.run(CommandPalette)
	ct.typ("upper")
	ct.key(0, ui.KeyEscape)
	if ct.st.ed.palette.open {
		t.Fatal("the palette stayed open")
	}
	ct.want("|aaa")
	if !ct.st.Focused() {
		t.Fatal("the keyboard did not come back to the editor")
	}
}

func TestCommandPaletteTitles(t *testing.T) {
	for cmd, want := range map[Command]string{
		NextDiagnostic:           "Next diagnostic",
		GoToLine:                 "Go to line",
		ToggleSoftWrap:           "Toggle soft wrap",
		Uppercase:                "Uppercase",
		FoldLevel1:               "Fold level1",
		CommandPalette:           "Command palette",
		SortLinesCaseInsensitive: "Sort lines case insensitive",
	} {
		if got := commandTitle(cmd); got != want {
			t.Fatalf("%v is titled %q, want %q", cmd, got, want)
		}
	}
}
