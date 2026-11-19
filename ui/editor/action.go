package editor

import (
	"slices"
)

// CodeAction is something done to the text to fix or change it: a quick
// fix of a diagnostic, or an organize imports, as a language server
// offers. The app gives them with ShowCodeActions.
type CodeAction struct {
	// Title is what the action is called, which the menu lists.
	Title string
	// Edits are what it does to the text: every change together, undone
	// as one step (ApplyEdits).
	Edits []Edit
}

// CodeActionRequest asks what can be done to the text at Offset, as the
// command CodeActions runs.
type CodeActionRequest struct {
	Offset  int
	Version uint64
}

// RenameRequest asks about renaming: with NewName empty, what the caret
// is on to rename, which ShowRename answers; with it, what to rename it
// to, whose edits the app applies with ApplyEdits.
type RenameRequest struct {
	Offset  int
	Version uint64
	// Start and End are the range being renamed, and OldName the text
	// there, as ShowRename gave them. NewName is what the bar typed.
	Start, End int
	OldName    string
	NewName    string
}

// codeActions is the menu of what can be done to the text at the caret.
type codeActions struct {
	shown   bool
	asked   int // the offset a request was made for
	actions []CodeAction
	at      int // where the menu sits
	chosen  int // the index in actions the arrows chose
	scrollTo bool
}

func (a *codeActions) open() bool { return a.shown }

func (a *codeActions) close() {
	a.shown = false
	a.actions = nil
}

// step moves the choice by d, wrapping around the actions, as the
// completions' do.
func (a *codeActions) step(d int) {
	if n := len(a.actions); n > 0 {
		a.chosen = (a.chosen + d + n) % n
		a.scrollTo = true
	}
}

// acting reports whether the menu of code actions is open, closing it
// when the caret moves, as completing does.
func (ed *editor) acting() bool {
	a := &ed.actions
	if p := ed.primary(); a.shown && p.head != a.at {
		a.close()
	}
	return a.shown
}

// acceptAction makes the edits of the action the menu chose.
func (ed *editor) acceptAction() bool {
	a := &ed.actions
	if !a.open() || a.chosen < 0 || a.chosen >= len(a.actions) {
		return false
	}
	act := a.actions[a.chosen]
	a.close()
	return ed.applyEdits(act.Edits)
}

// renameBar is the bar renaming the name it holds: the text typed in
// its field goes to OnRename as it is submitted.
type renameBar struct {
	open, focus bool
	start, end  int
	oldName     string
	text        string
}

func (r *renameBar) show(ed *editor, start, end int, oldName string) {
	r.open, r.focus = true, true
	r.start, r.end, r.oldName = start, end, oldName
	r.text = oldName
	ed.needsBuild = true
}

func (r *renameBar) close(ed *editor) {
	r.open = false
	ed.refocus = true
}

// submit asks the app to rename the name to what the bar typed.
func (r *renameBar) submit(ed *editor) {
	r.open = false
	ed.refocus = true
	if ed.o.onRename == nil {
		return
	}
	ed.o.onRename(RenameRequest{
		Offset: ed.primary().head, Version: ed.view.version,
		Start: r.start, End: r.end, OldName: r.oldName, NewName: r.text,
	})
}

// ApplyEdits makes the edits of a code action or a rename together, as
// one step of undo: the range each replaces, and what replaces it. It
// reports whether they did.
func (st *State) ApplyEdits(edits []Edit) bool {
	st.ensure()
	st.ed.sync()
	return st.ed.applyEdits(edits)
}

// applyEdits makes edits the app gave, in the order of the text.
func (ed *editor) applyEdits(edits []Edit) bool {
	if len(edits) == 0 {
		return false
	}
	all := slices.Clone(edits)
	slices.SortStableFunc(all, func(a, b Edit) int {
		if a.Start != b.Start {
			return a.Start - b.Start
		}
		return a.End - b.End
	})
	return ed.apply(all, editOther, nil)
}

// ShowCodeActions shows what can be done to the text, as a menu at the
// offset CodeActionRequest asked of. One chosen makes its edits with
// ApplyEdits.
func (st *State) ShowCodeActions(actions []CodeAction) {
	st.ensure()
	st.ed.sync()
	a := &st.ed.actions
	a.shown = len(actions) > 0
	a.actions = slices.Clone(actions)
	a.at = st.ed.primary().head
	a.chosen = 0
	a.scrollTo = true
	st.ed.needsBuild = true
}

// ShowRename renames the name from start to end: the bar types over
// oldName, and OnRename is asked with the new name as it is submitted.
func (st *State) ShowRename(start, end int, oldName string) {
	st.ensure()
	st.ed.sync()
	st.ed.rename.show(&st.ed, start, end, oldName)
}

// requestCodeActions asks the app what can be done at the newest caret.
func (ed *editor) requestCodeActions() bool {
	if ed.o.onCodeActions == nil {
		return false
	}
	off := ed.primary().head
	ed.actions.asked = off
	ed.o.onCodeActions(CodeActionRequest{Offset: off, Version: ed.view.version})
	return true
}

// requestRename asks the app what the newest caret is on to rename, or,
// without an app to ask, selects every occurrence of the word at the
// caret, which typing renames at once.
func (ed *editor) requestRename() bool {
	if ed.o.onRename == nil {
		a, z := wordAt(ed.view.text, ed.primary().head, ed.wordChars())
		if a >= z {
			return false
		}
		ed.setSels([]sel{ed.newSel(a, z)}, revealFit)
		return ed.selectAllMatches()
	}
	off := ed.primary().head
	ed.o.onRename(RenameRequest{Offset: off, Version: ed.view.version})
	return true
}
