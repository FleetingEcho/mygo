package editor

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
)

// The parts of an editor around its text: its find bar, its go to line
// bar, its command palette, its completions and its hovers. The editor
// places each, as an element with the behavior the part needs, and a
// render function styles and fills it, given a handle to what the part
// shows and does, as ui's bases are styled: the State's (RenderFindBar
// and the others), else the editor's own look (DefaultFindBar and the
// others), which an app's can call to wrap.

// FindBar is an editor's find bar, as RenderFindBar builds it: the query and
// its options, which the matches follow as they change, and what the bar
// does.
type FindBar struct {
	c  *ui.Context
	ed *editor
}

// Query and Replacement are the texts of the bar's fields, to bind inputs
// to, with QueryInput and ReplacementInput.
func (f *FindBar) Query() *string       { return &f.ed.find.query }
func (f *FindBar) Replacement() *string { return &f.ed.find.replace }

// CaseSensitive, WholeWord, Regexp and InSelection are the options of the
// search, to bind toggles to: matching case, whole words only, the query
// as a regular expression (RE2, as package regexp's), and only within the
// selections there were as the option was turned on.
func (f *FindBar) CaseSensitive() *bool { return &f.ed.find.caseSens }
func (f *FindBar) WholeWord() *bool     { return &f.ed.find.wholeWord }
func (f *FindBar) Regexp() *bool        { return &f.ed.find.regex }
func (f *FindBar) InSelection() *bool   { return &f.ed.find.inSelection }

// Replacing reports whether the bar replaces too, as Cmd+Alt+F opens it,
// and SetReplacing shows or hides replacing, which CanReplace reports a
// read-only editor never does.
func (f *FindBar) Replacing() bool      { return f.ed.find.replacing && !f.ed.o.readOnly }
func (f *FindBar) SetReplacing(on bool) { f.ed.find.replacing = on }
func (f *FindBar) CanReplace() bool     { return !f.ed.o.readOnly }

// Matches returns how many matches there are, and the index of the one
// selected, -1 without matches.
func (f *FindBar) Matches() (count, current int) {
	fd := &f.ed.find
	fd.search(f.ed)
	if len(fd.matches) == 0 {
		return 0, -1
	}
	return len(fd.matches), fd.current
}

// Status says what was found, as "3 of 12", "No results", or "Invalid"
// for a regular expression that is not one (Err); nothing without a
// query.
func (f *FindBar) Status() string { return f.ed.find.status(l10n.In(f.c.Locale())) }

// Err is the error of a query that is not a valid regular expression.
func (f *FindBar) Err() error { return f.ed.find.err }

// Next and Previous select the next and previous match after the caret,
// round the ends; SelectAll selects every match, with a caret each, and
// closes the bar.
func (f *FindBar) Next()      { f.ed.find.step(f.ed, 1) }
func (f *FindBar) Previous()  { f.ed.find.step(f.ed, -1) }
func (f *FindBar) SelectAll() { f.ed.find.selectAll(f.ed) }

// Replace replaces the match selected and selects the next; ReplaceAll
// replaces every match, as one step of undo. A regular expression's
// replacement expands $1 and ${name}.
func (f *FindBar) Replace()    { f.ed.find.replaceOne(f.ed) }
func (f *FindBar) ReplaceAll() { f.ed.find.replaceAll(f.ed) }

// Close closes the bar, and gives the keyboard back to the text.
func (f *FindBar) Close() {
	f.ed.find.open = false
	// The bar is gone: no match of it is on its way into view.
	f.ed.revealAt = -1
	f.ed.refocus = true
}

// QueryInput makes in the field of the query: it takes the focus with
// the query selected as the bar opens, and Enter goes to the next match,
// Shift+Enter to the previous, Alt+Enter selects them all, and Escape
// closes the bar. It returns in.
func (f *FindBar) QueryInput(in ui.Element) ui.Element {
	fd := &f.ed.find
	if fd.focusQuery {
		fd.focusQuery = false
		in.Focus().SetTextSelection(0, utf8.RuneCountInString(fd.query))
	}
	mods := f.c.Modifiers()
	switch {
	case in.Submitted() && mods&ui.Shift != 0:
		f.Previous()
	case in.Submitted() && mods&ui.Alt != 0:
		f.SelectAll()
	case in.Submitted():
		f.Next()
	case in.Shortcut(0, ui.KeyEscape):
		f.Close()
	}
	return in
}

// ReplacementInput makes in the field of the replacement: Enter replaces
// the match selected, Cmd+Enter all of them, and Escape closes the bar.
// It returns in.
func (f *FindBar) ReplacementInput(in ui.Element) ui.Element {
	switch {
	case in.Submitted() && f.c.Modifiers()&ui.Cmd != 0:
		f.ReplaceAll()
	case in.Submitted():
		f.Replace()
	case in.Shortcut(0, ui.KeyEscape):
		f.Close()
	}
	return in
}

const (
	findBarReduced   = 520
	findBarCollapsed = 360
)

// findBarSize is how much of the default find bar an editor w wide shows:
// 2 all of it, 1 all but the count, 0 the fields, the button showing or
// hiding the replacement and the close button.
func findBarSize(w float32) int {
	switch {
	case w < findBarCollapsed:
		return 0
	case w < findBarReduced:
		return 1
	}
	return 2
}

// DefaultFindBar builds the editor's own find bar into bar: the query
// with its options inside its field, the count of matches and the buttons
// going through them, and the replacement below it while replacing. In a
// narrow editor it leaves out the count, then all but the fields, the
// button showing or hiding the replacement and the close button.
func DefaultFindBar(c *ui.Context, bar ui.Element, f *FindBar) {
	t := c.Theme()
	tx := l10n.In(c.Locale())
	size := findBarSize(f.ed.fitWidth())
	bar.Gap(t.Space(1)).Padding(t.Space(1.5)).Background(t.Surface).Border(1, t.Border).Radius(t.Radius).
		Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.18))
	field := func(value *string, placeholder string, invalid bool, input func(ui.Element) ui.Element, inside func()) {
		box := ui.Row(c).AlignItems(ui.Center).MaxWidth(t.Space(75)).
			Padding(0, t.Space(0.5), 0, 0).Background(t.Background).Radius(t.Radius - 2)
		var in ui.Element
		box.Children(func() {
			in = input(ui.TextInputBase(c, value).Grow(1).Placeholder(placeholder).Padding(t.Space(1), t.Space(2)).
				FontSize(t.FontSize - 1).Label(placeholder).FocusRing(false).Invalid(invalid))
			if size > 0 && inside != nil {
				inside()
			}
		})
		// The border tells an invalid query, as ui's fields do, since a
		// narrow bar leaves out the status that says so.
		border := t.Border
		switch {
		case invalid:
			border = t.Danger
		case in.Focused():
			border = t.Accent
		}
		box.Border(1, border)
	}
	button := func(label, tip string) ui.Element {
		b := ui.ButtonBase(c).Padding(t.Space(0.5), t.Space(1.5)).Radius(t.Radius - 2).Label(tip).Tooltip(tip).KeepFocus()
		b.Children(func() { ui.Text(c, label).FontSize(t.FontSize - 1) })
		return b
	}
	toggle := func(on *bool, label, tip string) {
		b := ui.ToggleBase(c, on).Padding(t.Space(0.5), t.Space(1)).Radius(t.Radius - 2).Label(tip).Tooltip(tip).KeepFocus()
		if *on {
			b.Background(t.Accent.Alpha(0.2)).TextColor(t.Accent)
		}
		b.Children(func() { ui.Text(c, label).FontSize(t.FontSize - 2) })
	}
	row := func(children func()) {
		ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(children)
	}
	// The fields in the first column, as wide as each other, their buttons
	// in the second.
	bar.Grid().ColumnTracks(ui.Fr(1), ui.FitContent()).Children(func() {
		field(f.Query(), tx.Find(), f.Err() != nil, f.QueryInput, func() {
			toggle(f.CaseSensitive(), "Aa", tx.MatchCase())
			toggle(f.WholeWord(), "ab", tx.MatchWholeWord())
			toggle(f.Regexp(), ".*", tx.UseRegularExpression())
			toggle(f.InSelection(), "[ ]", tx.FindInSelection())
		})
		row(func() {
			if size > 1 {
				ui.Text(c, f.Status()).FontSize(t.FontSize - 2).TextColor(t.TextMuted).FontFeatures("tnum").MinWidth(t.Space(16)).Role(ui.RoleStatus)
			}
			if size > 0 {
				if button("↑", tx.PreviousMatch()).Clicked() {
					f.Previous()
				}
				if button("↓", tx.NextMatch()).Clicked() {
					f.Next()
				}
			}
			if f.CanReplace() && button("⇄", tx.Replace()).Clicked() {
				f.SetReplacing(!f.Replacing())
			}
			if button("✕", tx.Close()).Clicked() {
				f.Close()
			}
		})
		if f.Replacing() {
			field(f.Replacement(), tx.Replace(), false, f.ReplacementInput, nil)
			row(func() {
				if size > 0 {
					if ui.Button(c, tx.Replace()).Padding(t.Space(0.5), t.Space(2)).FontSize(t.FontSize - 1).Clicked() {
						f.Replace()
					}
					if ui.Button(c, tx.ReplaceAll()).Padding(t.Space(0.5), t.Space(2)).FontSize(t.FontSize - 1).Clicked() {
						f.ReplaceAll()
					}
				}
			})
		}
	})
}

// barInRow builds the row a bar of a viewer as high as its lines sticks
// in: over the whole of it, with the bar at the top of the part of the
// text the containers around it show, which they take out of view
// otherwise. justify places the bar across the viewer; newBar makes it,
// as wide as its content, and build fills and renders it within the row,
// as an app's render function may add elements beside the bar's own. An
// editor that scrolls itself places its bars as it did: the find bar an
// absolute box at its top right, Go to Line centered at its top.
func (ed *editor) barInRow(c *ui.Context, rowKey string, justify ui.Align, newBar func() ui.Element, build func(ui.Element)) {
	t := c.Theme()
	row := ui.Row(c).Key(rowKey).Absolute().Top(0).Bottom(0).Left(0).Right(0).
		Justify(justify).PassThrough()
	row.Children(func() {
		build(newBar().Sticky().Top(t.Space(1)).AlignSelf(ui.Start))
	})
}

// buildFindBar places the find bar over the editor's top right corner,
// no wider than the editor, and builds it. Its margins, unlike its insets,
// narrow the room it fits its content in.
func (ed *editor) buildFindBar(c *ui.Context) {
	f := &ed.find
	f.update(ed)
	t := c.Theme()
	f.builtW = ed.fitWidth()
	render := ed.o.renderFindBar
	if render == nil {
		render = DefaultFindBar
	}
	newBar := func() ui.Element {
		return ui.Box(c).Key("code find").Role(ui.RoleToolbar).Label(l10n.In(c.Locale()).Find()).
			Cursor(ui.CursorDefault).HandleInput(shield).Margin(0, t.Space(4), 0, t.Space(1))
	}
	if ed.o.fitContent {
		ed.barInRow(c, "code find row", ui.End, newBar, func(bar ui.Element) {
			render(c, bar, &FindBar{c: c, ed: ed})
		})
	} else {
		// An editor that scrolls itself places its bar as it always has:
		// absolute, so insets an app's render function sets place it.
		render(c, newBar().Absolute().Top(t.Space(1)).Right(0), &FindBar{c: c, ed: ed})
	}
	// What the bar typed or toggled shows at once.
	f.update(ed)
	if isMac() {
		switch {
		case c.Shortcut(ui.Cmd, ui.KeyG):
			f.step(ed, 1)
		case c.Shortcut(ui.Shift|ui.Cmd, ui.KeyG):
			f.step(ed, -1)
		}
	} else {
		switch {
		case c.Shortcut(0, ui.KeyF3):
			f.step(ed, 1)
		case c.Shortcut(ui.Shift, ui.KeyF3):
			f.step(ed, -1)
		}
	}
}

// GoToLineBar is an editor's go to line bar, as RenderGoToLine builds it:
// line[:column], or +n and -n lines from the caret, which the editor
// scrolls to as it is typed, as Zed's.
type GoToLineBar struct {
	c  *ui.Context
	ed *editor
}

// Text is the text of the bar's field, to bind an input to, with Input.
func (g *GoToLineBar) Text() *string { return &g.ed.goTo.text }

// Input makes in the bar's field: it takes the focus as the bar opens,
// Enter goes to the line, and Escape closes the bar where the editor
// was. It returns in.
func (g *GoToLineBar) Input(in ui.Element) ui.Element {
	gt := &g.ed.goTo
	if gt.focus {
		gt.focus = false
		in.Focus()
	}
	switch {
	case in.Submitted():
		g.Go()
	case in.Shortcut(0, ui.KeyEscape):
		g.Close()
	}
	return in
}

// Cursor is where the newest caret is, and LineCount how many lines there
// are, as the bar tells.
func (g *GoToLineBar) Cursor() Pos    { return posOf(g.ed.view.text, g.ed.primary().head) }
func (g *GoToLineBar) LineCount() int { return g.ed.view.text.LineCount() }

// Target is the line and column typed, from 0, and whether the text is
// one.
func (g *GoToLineBar) Target() (Pos, bool) {
	line, col, ok := g.ed.goTo.target(g.ed)
	return Pos{line, col}, ok
}

// Go moves the caret to the line typed and closes the bar; Close closes
// it where the editor was.
func (g *GoToLineBar) Go() {
	ed := g.ed
	if line, col, ok := ed.goTo.target(ed); ok {
		off := offsetOf(ed.view.text, Pos{line, col})
		ed.goTo.open = false
		ed.setSels([]sel{ed.newSel(off, off)}, revealCenter)
		// The containers around a viewer as high as its lines scroll to
		// the line; the viewer itself has no scroll of its own.
		ed.askRevealNow(off)
		ed.refocus = true
	}
}
func (g *GoToLineBar) Close() { g.ed.goTo.close(g.ed) }

// DefaultGoToLine builds the editor's own go to line bar into bar: the
// field, and where the caret is.
func DefaultGoToLine(c *ui.Context, bar ui.Element, g *GoToLineBar) {
	t := c.Theme()
	tx := l10n.In(c.Locale())
	bar.Row().Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(1.5), t.Space(2)).Background(t.Surface).
		Border(1, t.Border).Radius(t.Radius).Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.18))
	bar.Children(func() {
		g.Input(ui.TextInputBase(c, g.Text()).Width(t.Space(60)).Padding(t.Space(1), t.Space(2)).Background(t.Background).
			Border(1, t.Border).Radius(t.Radius - 2).Label(tx.GoToLine()).Placeholder(tx.LineRange(g.LineCount())))
		cur := g.Cursor()
		ui.Text(c, tx.CurrentLine(cur.Line+1, g.LineCount(), cur.Col+1)).
			FontSize(t.FontSize - 2).TextColor(t.TextMuted)
	})
}

// buildGoToLine places the go to line bar in the middle of the editor's
// top, builds it, and shows the line typed.
func (ed *editor) buildGoToLine(c *ui.Context) {
	g := &ed.goTo
	render := ed.o.renderGoToLine
	if render == nil {
		render = DefaultGoToLine
	}
	newBar := func() ui.Element {
		return ui.Box(c).Role(ui.RoleDialog).Label(l10n.In(c.Locale()).GoToLine()).
			Cursor(ui.CursorDefault).HandleInput(shield)
	}
	if ed.o.fitContent {
		ed.barInRow(c, "code go to line", ui.Center, newBar, func(bar ui.Element) {
			render(c, bar, &GoToLineBar{c: c, ed: ed})
		})
	} else {
		// An editor that scrolls itself places its bar as it always has: a
		// row at its top, the bar centered in it.
		ui.Row(c).Key("code go to line").Absolute().Top(c.Theme().Space(1)).Left(0).Right(0).
			Justify(ui.Center).PassThrough().Children(func() {
			render(c, newBar(), &GoToLineBar{c: c, ed: ed})
		})
	}
	if !g.open {
		return
	}
	if line, col, ok := g.target(ed); ok && line != g.preview {
		g.preview = line
		// A viewer as high as its lines cannot scroll itself, and the
		// containers around it are not the editor's to move while the
		// user types: Escape, which puts back the editor's own scroll,
		// would leave them where the preview put them. Go takes the line
		// into view.
		if cr, ok := ed.caretRect(offsetOf(ed.view.text, Pos{line, col})); ok && !ed.o.fitContent {
			ed.setScroll(float64(cr.Y) - float64(ed.viewH-min(cr.H, ed.viewH))/2)
		}
	}
}

// PaletteBar is an editor's command palette, as RenderPalette builds it:
// every command of the editor, fuzzy matched to the text typed, and the
// one chosen, which Enter or a click runs.
type PaletteBar struct {
	c  *ui.Context
	ed *editor
	// chosen and closing are what the bar was asked while it was built,
	// done once it is, so that the rows after the one clicked still
	// build: the command chosen to run, -1 for none.
	chosen  int
	closing bool
}

// Text is the text of the bar's field, to bind an input to, with Input.
func (p *PaletteBar) Text() *string { return &p.ed.palette.text }

// Input makes in the bar's field: it takes the focus as the bar opens,
// the arrows choose a command, Enter runs the one chosen, and Escape
// closes the bar back to the editor. It returns in.
func (p *PaletteBar) Input(in ui.Element) ui.Element {
	pal := &p.ed.palette
	if pal.focus {
		pal.focus = false
		in.Focus()
	}
	switch {
	case in.Shortcut(0, ui.KeyDown):
		pal.step(1)
	case in.Shortcut(0, ui.KeyUp):
		pal.step(-1)
	case in.Submitted():
		p.Choose(pal.chosen)
	case in.Shortcut(0, ui.KeyEscape):
		p.Close()
	}
	return in
}

// Items are the commands shown, best first: those the text typed
// matches, or every command in its order while the text is empty.
func (p *PaletteBar) Items() []Command { return slices.Clone(p.ed.palette.matches) }

// Selected is the index in Items of the command chosen.
func (p *PaletteBar) Selected() int { return p.ed.palette.chosen }

// Command is the command of Items at i, and Name its name as the palette
// lists it, as "Next diagnostic" for nextDiagnostic.
func (p *PaletteBar) Command(i int) Command { return p.ed.palette.matches[i] }

func (p *PaletteBar) Name(i int) string { return commandTitle(p.Command(i)) }

// Shortcut is the key that runs the command, as menus show it, or "" for
// none. The app's bindings come before the editor's own.
func (p *PaletteBar) Shortcut(i int) string {
	cmd := p.Command(i)
	for _, list := range [][]Binding{p.ed.o.bindings, defaultCodeBindings()} {
		for j := range list {
			b := &list[j]
			if b.Command != cmd {
				continue
			}
			text := ui.ShortcutText(b.Mods, b.Key)
			if b.ThenKey != 0 {
				text += " " + ui.ShortcutText(b.ThenMods, b.ThenKey)
			}
			return text
		}
	}
	return ""
}

// Item makes the row of the command i of Items: selected while chosen,
// scrolled into view as the arrows choose it, and running it when
// clicked. Style it and build its content as the row of a list.
func (p *PaletteBar) Item(i int) ui.Element {
	pal := &p.ed.palette
	row := ui.ButtonBase(p.c).Key(i).FocusRing(false).Role(ui.RoleListItem).Label(p.Name(i)).Selected(i == pal.chosen)
	if i == pal.chosen && pal.scrollTo {
		row.ScrollIntoView()
	}
	if row.Clicked() {
		p.Choose(i)
	}
	return row
}

// Choose runs the command i of Items and closes the bar; Close closes it
// without running one. Both happen once the bar is built.
func (p *PaletteBar) Choose(i int) {
	if i >= 0 && i < len(p.ed.palette.matches) {
		p.chosen = i
	}
}

func (p *PaletteBar) Close() { p.closing = true }

// Font is the editor's font, LineHeight the height of its lines, and
// Theme its colors, to show the commands as the completions show theirs.
func (p *PaletteBar) Font() ui.Font       { return p.ed.view.font }
func (p *PaletteBar) LineHeight() float32 { return p.ed.view.lineHeight }
func (p *PaletteBar) Theme() *Theme       { return p.ed.theme }

// commandTitle writes a command's name as the palette lists it, as
// "Next diagnostic" for nextDiagnostic.
func commandTitle(c Command) string {
	name := c.String()
	var b strings.Builder
	for i, r := range name {
		if i > 0 && unicode.IsUpper(r) {
			b.WriteByte(' ')
		}
		if i == 0 {
			r = unicode.ToUpper(r)
		} else {
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

// maxListRows is how many rows of a list the default palette and the
// default go to symbol bar show.
const maxListRows = 200

// DefaultPalette builds the editor's own command palette into bar, as
// Zed's: the field matching commands as one types, and the list of them
// below it, nine rows high, each with the key that runs it.
func DefaultPalette(c *ui.Context, bar ui.Element, p *PaletteBar) {
	t := c.Theme()
	tx := l10n.In(c.Locale())
	font := p.Font()
	lh := p.LineHeight()
	bar.Column().Gap(t.Space(1)).Padding(t.Space(1.5)).Background(t.Surface).
		Border(1, t.Border).Radius(t.Radius).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.18)).Width(t.Space(80))
	bar.Children(func() {
		p.Input(ui.TextInputBase(c, p.Text()).Padding(t.Space(1), t.Space(2)).Background(t.Background).
			Border(1, t.Border).Radius(t.Radius - 2).Label(tx.CommandPalette()).Placeholder(tx.CommandPalette()).FocusRing(false))
		items := p.Items()
		if len(items) == 0 {
			return
		}
		ui.Scroll(c).Key("rows").MaxHeight(lh * 9).Children(func() {
			for i := range items {
				if i >= maxListRows {
					break
				}
				row := p.Item(i).Row().Justify(ui.Start).Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(0.5), t.Space(1.5)).
					Radius(t.Radius - 2).Height(lh + 2)
				if i == p.Selected() {
					row.Background(t.Accent.Alpha(0.18))
				}
				row.Children(func() {
					ui.Text(c, p.Name(i)).Font(font.Family).FontSize(font.Size).SingleLine().Shrink(1)
					if s := p.Shortcut(i); s != "" {
						ui.Text(c, s).FontSize(font.Size - 1).TextColor(t.TextMuted).SingleLine().Shrink(1).Grow(1).TextAlign(ui.End)
					}
				})
			}
		})
	})
}

// buildPalette places the command palette at the editor's top, above
// the window's content, and runs the command chosen.
func (ed *editor) buildPalette(c *ui.Context) {
	pal := &ed.palette
	if pal.queried != pal.text {
		pal.search()
	}
	t := c.Theme()
	anchor := ui.Box(c).Key("code palette anchor").Absolute().Top(t.Space(1)).Left(0).Right(0).Height(0).PassThrough()
	ui.Overlay(c, func() {
		bar := ui.Box(c).Key("code palette").AttachTo(anchor, ui.AnchorBottom, ui.AnchorTop).
			Role(ui.RoleDialog).Label(l10n.In(c.Locale()).CommandPalette()).Cursor(ui.CursorDefault).HandleInput(shield)
		render := ed.o.renderPalette
		if render == nil {
			render = DefaultPalette
		}
		p := &PaletteBar{c: c, ed: ed, chosen: -1}
		render(c, bar, p)
		switch {
		case p.chosen >= 0 && p.chosen < len(pal.matches):
			cmd := pal.matches[p.chosen]
			pal.close(ed)
			ed.run(cmd)
			c.Invalidate()
		case p.closing:
			pal.close(ed)
			c.Invalidate()
		}
	})
	pal.scrollTo = false
}

// SymbolBar is an editor's bar going to a symbol, as RenderSymbols
// builds it: the symbols of the State's Symbols fuzzy matched to the
// text typed, and the one chosen, which Enter or a click goes to.
type SymbolBar struct {
	c  *ui.Context
	ed *editor
	// chosen and closing are what the bar was asked while it was built,
	// done once it is, so that the rows after the one clicked still
	// build: the symbol chosen to go to, -1 for none.
	chosen  int
	closing bool
}

// Text is the text of the bar's field, to bind an input to, with Input.
func (s *SymbolBar) Text() *string { return &s.ed.symbols.text }

// Input makes in the bar's field: it takes the focus as the bar opens,
// the arrows choose a symbol, Enter goes to the one chosen, and Escape
// closes the bar back to the editor. It returns in.
func (s *SymbolBar) Input(in ui.Element) ui.Element {
	sym := &s.ed.symbols
	if sym.focus {
		sym.focus = false
		in.Focus()
	}
	switch {
	case in.Shortcut(0, ui.KeyDown):
		sym.step(1)
	case in.Shortcut(0, ui.KeyUp):
		sym.step(-1)
	case in.Submitted():
		s.Choose(sym.chosen)
	case in.Shortcut(0, ui.KeyEscape):
		s.Close()
	}
	return in
}

// Items are the symbols shown, best first: those the text typed
// matches, or every symbol in the order of the text while it is empty.
func (s *SymbolBar) Items() []Symbol { return slices.Clone(s.ed.symbols.matches) }

// Selected is the index in Items of the symbol chosen.
func (s *SymbolBar) Selected() int { return s.ed.symbols.chosen }

// Name is the name of the symbol i of Items, and Range where it is in
// the text.
func (s *SymbolBar) Name(i int) string { return s.ed.symbols.matches[i].Name }

func (s *SymbolBar) Range(i int) (start, end int) {
	sym := s.ed.symbols.matches[i]
	return sym.Start, sym.End
}

// Item makes the row of the symbol i of Items: selected while chosen,
// scrolled into view as the arrows choose it, and going to it when
// clicked. Style it and build its content as the row of a list.
func (s *SymbolBar) Item(i int) ui.Element {
	sym := &s.ed.symbols
	row := ui.ButtonBase(s.c).Key(i).FocusRing(false).Role(ui.RoleListItem).Label(s.Name(i)).Selected(i == sym.chosen)
	if i == sym.chosen && sym.scrollTo {
		row.ScrollIntoView()
	}
	if row.Clicked() {
		s.Choose(i)
	}
	return row
}

// Choose goes to the symbol i of Items and closes the bar; Close closes
// it without going to one. Both happen once the bar is built.
func (s *SymbolBar) Choose(i int) {
	if i >= 0 && i < len(s.ed.symbols.matches) {
		s.chosen = i
	}
}

func (s *SymbolBar) Close() { s.closing = true }

// Font is the editor's font, LineHeight the height of its lines, and
// Theme its colors, to show the symbols as the completions show theirs.
func (s *SymbolBar) Font() ui.Font       { return s.ed.view.font }
func (s *SymbolBar) LineHeight() float32 { return s.ed.view.lineHeight }
func (s *SymbolBar) Theme() *Theme       { return s.ed.theme }

// DefaultSymbols builds the editor's own bar going to a symbol into
// bar, as VS Code's: the field matching symbols as one types, and the
// list of them below it, nine rows high.
func DefaultSymbols(c *ui.Context, bar ui.Element, s *SymbolBar) {
	t := c.Theme()
	tx := l10n.In(c.Locale())
	font := s.Font()
	lh := s.LineHeight()
	bar.Column().Gap(t.Space(1)).Padding(t.Space(1.5)).Background(t.Surface).
		Border(1, t.Border).Radius(t.Radius).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.18)).Width(t.Space(80))
	bar.Children(func() {
		s.Input(ui.TextInputBase(c, s.Text()).Padding(t.Space(1), t.Space(2)).Background(t.Background).
			Border(1, t.Border).Radius(t.Radius - 2).Label(tx.GoToSymbol()).Placeholder(tx.GoToSymbol()).FocusRing(false))
		items := s.Items()
		if len(items) == 0 {
			return
		}
		ui.Scroll(c).Key("rows").MaxHeight(lh * 9).Children(func() {
			for i := range items {
				if i >= maxListRows {
					break
				}
				row := s.Item(i).Row().AlignItems(ui.Center).Padding(t.Space(0.5), t.Space(1.5)).
					Radius(t.Radius - 2).Height(lh + 2)
				if i == s.Selected() {
					row.Background(t.Accent.Alpha(0.18))
				}
				row.Children(func() {
					ui.Text(c, s.Name(i)).Font(font.Family).FontSize(font.Size).SingleLine().Shrink(1)
				})
			}
		})
	})
}

// buildSymbols places the bar going to a symbol at the editor's top,
// above the window's content, and goes to the symbol chosen.
func (ed *editor) buildSymbols(c *ui.Context) {
	sym := &ed.symbols
	if sym.queried != sym.text {
		sym.search(ed)
	}
	t := c.Theme()
	anchor := ui.Box(c).Key("code symbols anchor").Absolute().Top(t.Space(1)).Left(0).Right(0).Height(0).PassThrough()
	ui.Overlay(c, func() {
		bar := ui.Box(c).Key("code symbols").AttachTo(anchor, ui.AnchorBottom, ui.AnchorTop).
			Role(ui.RoleDialog).Label(l10n.In(c.Locale()).GoToSymbol()).Cursor(ui.CursorDefault).HandleInput(shield)
		render := ed.o.renderSymbols
		if render == nil {
			render = DefaultSymbols
		}
		s := &SymbolBar{c: c, ed: ed, chosen: -1}
		render(c, bar, s)
		switch {
		case s.chosen >= 0 && s.chosen < len(sym.matches):
			target := sym.matches[s.chosen]
			sym.close(ed)
			off := min(max(0, target.Start), ed.view.text.Len())
			ed.setSels([]sel{ed.newSel(off, off)}, revealCenter)
			c.Invalidate()
		case s.closing:
			sym.close(ed)
			c.Invalidate()
		}
	})
	sym.scrollTo = false
}

// CompletionMenu is an editor's list of completions, as
// RenderCompletions builds it: the completions matching the word typed,
// best first, and the one chosen, which the arrows move and Enter or Tab
// inserts.
type CompletionMenu struct {
	c  *ui.Context
	ed *editor
	// chosen and closing are what the list was asked while it was built,
	// done once it is, so that the rows after the one clicked still
	// build: the item chosen, -1 for none.
	chosen  int
	closing bool
}

// Items returns the completions shown, best first.
func (m *CompletionMenu) Items() []Completion {
	cc := &m.ed.complete
	out := make([]Completion, len(cc.filtered))
	for i, idx := range cc.filtered {
		out[i] = cc.items[idx]
	}
	return out
}

// Selected is the index in Items of the completion chosen.
func (m *CompletionMenu) Selected() int { return m.ed.complete.selected }

// Prefix is the word typed, which the completions match.
func (m *CompletionMenu) Prefix() string { return m.ed.complete.prefix }

// Item makes the row of completion i of Items: selected while chosen,
// scrolled into view as the arrows choose it, and inserting it when
// clicked. Style it and build its content as the row of a list.
func (m *CompletionMenu) Item(i int) ui.Element {
	cc := &m.ed.complete
	it := cc.items[cc.filtered[i]]
	row := ui.ButtonBase(m.c).Key(i).FocusRing(false).Role(ui.RoleListItem).Label(it.Label).Selected(i == cc.selected)
	if i == cc.selected && cc.scrollTo {
		row.ScrollIntoView()
	}
	if row.Clicked() {
		m.Choose(i)
	}
	return row
}

// Choose inserts completion i of Items in place of the word typed, and
// closes the list; Close closes it. Both happen once the list is built.
func (m *CompletionMenu) Choose(i int) {
	if i >= 0 && i < len(m.ed.complete.filtered) {
		m.chosen = i
	}
}
func (m *CompletionMenu) Close() { m.closing = true }

// Font is the editor's font, LineHeight the height of its lines, and
// Theme its colors, to show code as the editor does.
func (m *CompletionMenu) Font() ui.Font       { return m.ed.view.font }
func (m *CompletionMenu) LineHeight() float32 { return m.ed.view.lineHeight }
func (m *CompletionMenu) Theme() *Theme       { return m.ed.theme }

// Icon returns a letter or symbol for a kind of completion, and Token the
// kind of token it is, whose color tints it, as the default list shows.
func (k CompletionKind) Icon() string {
	switch k {
	case CompletionKeyword:
		return "k"
	case CompletionFunction, CompletionMethod:
		return "ƒ"
	case CompletionVariable:
		return "x"
	case CompletionField:
		return "·"
	case CompletionType:
		return "T"
	case CompletionModule:
		return "{}"
	case CompletionConstant:
		return "π"
	case CompletionSnippet:
		return "⌘"
	}
	return "≡"
}

func (k CompletionKind) Token() TokenKind {
	switch k {
	case CompletionKeyword:
		return TokenKeyword
	case CompletionFunction, CompletionMethod:
		return TokenFunction
	case CompletionField:
		return TokenProperty
	case CompletionType, CompletionModule:
		return TokenType
	case CompletionConstant:
		return TokenConstant
	}
	return TokenText
}

// maxCompletionRows is how many completions the default list shows.
const maxCompletionRows = 200

// DefaultCompletions builds the editor's own list of completions into
// panel, as Zed's: each with its kind, label and detail, nine rows high,
// and the documentation of the one chosen below them.
func DefaultCompletions(c *ui.Context, panel ui.Element, m *CompletionMenu) {
	t := c.Theme()
	ct := m.Theme()
	font := m.Font()
	lh := m.LineHeight()
	panel.Background(t.Surface).Border(1, t.Border).Radius(t.Radius).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.2)).
		Padding(t.Space(1)).MaxHeight(lh*9 + t.Space(2)).MinWidth(t.Space(60)).MaxWidth(t.Space(150))
	items := m.Items()
	panel.Children(func() {
		ui.Scroll(c).Key("rows").MaxHeight(lh * 9).Children(func() {
			for i, it := range items {
				if i >= maxCompletionRows {
					break
				}
				row := m.Item(i).Row().Justify(ui.Start).Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(0.5), t.Space(1.5)).
					Radius(t.Radius - 2).Height(lh + 2)
				if i == m.Selected() {
					row.Background(t.Accent.Alpha(0.18))
				}
				row.Children(func() {
					ui.Text(c, it.Kind.Icon()).Font("monospace").FontSize(font.Size - 1).TextColor(ct.TokenColor(it.Kind.Token())).Width(t.Space(3))
					ui.Text(c, it.Label).Font(font.Family).FontSize(font.Size).TextColor(ct.Text).SingleLine().Shrink(1)
					if it.Detail != "" {
						ui.Text(c, it.Detail).FontSize(font.Size - 1).TextColor(t.TextMuted).SingleLine().Shrink(1).Grow(1).TextAlign(ui.End)
					}
				})
			}
		})
		if s := m.Selected(); s < len(items) && items[s].Documentation != "" {
			ui.Box(c).Height(1).Background(t.Border).Margin(t.Space(1), 0)
			ui.Text(c, items[s].Documentation).FontSize(t.FontSize-1).TextColor(t.TextMuted).MaxWidth(t.Space(140)).Padding(t.Space(0.5), t.Space(1.5))
		}
	})
}

// buildCompletions places the list of completions below the word being
// completed, above the window's content, and builds it.
func (ed *editor) buildCompletions(c *ui.Context, anchor ui.Element) {
	cc := &ed.complete
	ui.Overlay(c, func() {
		panel := ui.Box(c).Key("code completions").AttachTo(anchor, ui.AnchorBottomLeft, ui.AnchorTopLeft).
			Role(ui.RoleList).Label(l10n.In(c.Locale()).Completions()).Cursor(ui.CursorDefault).KeepFocus().HandleInput(shield)
		render := ed.o.renderCompletions
		if render == nil {
			render = DefaultCompletions
		}
		m := &CompletionMenu{c: c, ed: ed, chosen: -1}
		render(c, panel, m)
		switch {
		case m.chosen >= 0:
			cc.selected = m.chosen
			ed.acceptCompletion()
			c.Invalidate()
		case m.closing:
			cc.close()
			c.Invalidate()
		}
	})
	cc.scrollTo = false
}

// HoverCard is what an editor shows of the text under the pointer, as
// RenderHover builds it: the diagnostics there, and what the app gave
// ShowHover.
type HoverCard struct {
	c      *ui.Context
	ed     *editor
	anchor ui.Handle
}

// Anchor resolves the hovered source range's element in the current build.
// A custom renderer can reattach a large card so it remains inside the window.
// Like other build elements, it must not be retained after rendering.
func (h *HoverCard) Anchor() ui.Element { return h.c.Resolve(h.anchor) }

// Diagnostics are the diagnostics of the text hovered.
func (h *HoverCard) Diagnostics() []Diagnostic { return h.ed.hover.diags }

// Text is the text ShowHover gave, and Content builds what its function
// builds, when it gave one.
func (h *HoverCard) Text() string { return h.ed.hover.text }
func (h *HoverCard) Content() {
	if b := h.ed.hover.build; b != nil {
		b(h.c)
	}
}

// Range is the text hovered, from start to end.
func (h *HoverCard) Range() (start, end int) { return h.ed.hover.start, h.ed.hover.end }

// Font is the editor's font, and Theme its colors.
func (h *HoverCard) Font() ui.Font { return h.ed.view.font }
func (h *HoverCard) Theme() *Theme { return h.ed.theme }

// DefaultHover builds the editor's own hover into panel, as Zed's: the
// diagnostics with bars of their severity's color, then the text in the
// editor's font and what the app builds, all of it selectable.
func DefaultHover(c *ui.Context, panel ui.Element, h *HoverCard) {
	t := c.Theme()
	ct := h.Theme()
	panel.Background(t.Surface).Border(1, t.Border).Radius(t.Radius).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.2)).
		Padding(t.Space(2)).Gap(t.Space(1.5)).MaxWidth(t.Space(150)).Margin(0, 0, t.Space(1), 0).Selectable()
	panel.Children(func() {
		for _, d := range h.Diagnostics() {
			ui.Row(c).Gap(t.Space(1.5)).Children(func() {
				ui.Box(c).Width(3).Radius(1.5).Background(ct.SeverityColor(d.Severity)).AlignSelf(ui.Stretch)
				msg := d.Message
				if d.Source != "" {
					msg += " (" + d.Source + ")"
				}
				ui.Text(c, msg).FontSize(t.FontSize - 1).Shrink(1)
			})
		}
		if text := h.Text(); text != "" {
			ui.Text(c, text).Font(h.Font().Family).FontSize(h.Font().Size).TextColor(ct.Text)
		}
		h.Content()
	})
}

// buildHover places the hover above the text hovered, above the window's
// content, and builds it.
func (ed *editor) buildHover(c *ui.Context, anchor ui.Element) {
	h := &ed.hover
	if !h.shown || h.build == nil && h.text == "" && len(h.diags) == 0 {
		return
	}
	ui.Overlay(c, func() {
		// The hover takes the keyboard as it is pressed, for its text to
		// be selected and copied; Escape gives it back to the editor.
		panel := ui.Box(c).Key("code hover").AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).
			Role(ui.RoleTooltip).HandleInput(shield)
		h.over, h.focused = panel.Hovered(), panel.FocusWithin()
		if panel.Shortcut(0, ui.KeyEscape) {
			ed.closeHover()
		}
		render := ed.o.renderHover
		if render == nil {
			render = DefaultHover
		}
		card := &HoverCard{c: c, ed: ed}
		anchor = anchor.Bind(&card.anchor)
		render(c, panel, card)
	})
}

// SignatureCard is the signature of the call the caret is in, as
// RenderSignature builds it.
type SignatureCard struct {
	c      *ui.Context
	ed     *editor
	anchor ui.Handle
}

// Anchor resolves the call's element in the current build. A custom
// renderer can reattach a large card so it remains inside the window.
// Like other build elements, it must not be retained after rendering.
func (s *SignatureCard) Anchor() ui.Element { return s.c.Resolve(s.anchor) }

// Text is the text ShowSignatureHelp gave, and Content builds what its
// function builds, when it gave one.
func (s *SignatureCard) Text() string { return s.ed.sig.text }

func (s *SignatureCard) Content() {
	if b := s.ed.sig.build; b != nil {
		b(s.c)
	}
}

// Range is the call's text, from start to end.
func (s *SignatureCard) Range() (start, end int) { return s.ed.sig.start, s.ed.sig.end }

// Font is the editor's font, and Theme its colors.
func (s *SignatureCard) Font() ui.Font { return s.ed.view.font }
func (s *SignatureCard) Theme() *Theme { return s.ed.theme }

// DefaultSignature builds the editor's own signature card into panel:
// the text in the editor's font and what the app builds, all of it
// selectable.
func DefaultSignature(c *ui.Context, panel ui.Element, s *SignatureCard) {
	t := c.Theme()
	ct := s.Theme()
	panel.Background(t.Surface).Border(1, t.Border).Radius(t.Radius).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.2)).
		Padding(t.Space(1.5)).MaxWidth(t.Space(150)).Margin(0, 0, t.Space(1), 0).Selectable()
	panel.Children(func() {
		if text := s.Text(); text != "" {
			ui.Text(c, text).Font(s.Font().Family).FontSize(s.Font().Size).TextColor(ct.Text)
		}
		s.Content()
	})
}

// buildSignature places the signature of the call above its start,
// above the window's content, and builds it.
func (ed *editor) buildSignature(c *ui.Context, anchor ui.Element) {
	s := &ed.sig
	// The card goes when the caret leaves the call it shows.
	if off := ed.primary().head; off < s.start || off > s.end {
		s.close()
		c.Invalidate()
		return
	}
	ui.Overlay(c, func() {
		panel := ui.Box(c).Key("code signature").AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).
			Role(ui.RoleTooltip).HandleInput(shield)
		s.focused = panel.FocusWithin()
		if panel.Shortcut(0, ui.KeyEscape) {
			s.close()
			ed.refocus = true
		}
		render := ed.o.renderSignature
		if render == nil {
			render = DefaultSignature
		}
		card := &SignatureCard{c: c, ed: ed}
		anchor = anchor.Bind(&card.anchor)
		render(c, panel, card)
	})
}

// ActionMenu is an editor's menu of what can be done to the text at the
// caret, as RenderActions builds it: the CodeActions the app gave, and
// the one chosen, which Enter or a click makes the edits of.
type ActionMenu struct {
	c  *ui.Context
	ed *editor
	// chosen and closing are what the menu was asked while it was built,
	// done once it is, so that the rows after the one clicked still
	// build: the action chosen, -1 for none.
	chosen  int
	closing bool
}

// Items are the actions shown.
func (m *ActionMenu) Items() []CodeAction { return slices.Clone(m.ed.actions.actions) }

// Selected is the index in Items of the action chosen.
func (m *ActionMenu) Selected() int { return m.ed.actions.chosen }

// Title is what the action i of Items is called.
func (m *ActionMenu) Title(i int) string { return m.ed.actions.actions[i].Title }

// Item makes the row of the action i of Items: selected while chosen,
// scrolled into view as the arrows choose it, and making it when
// clicked. Style it and build its content as the row of a list.
func (m *ActionMenu) Item(i int) ui.Element {
	a := &m.ed.actions
	row := ui.ButtonBase(m.c).Key(i).FocusRing(false).Role(ui.RoleListItem).Label(m.Title(i)).Selected(i == a.chosen)
	if i == a.chosen && a.scrollTo {
		row.ScrollIntoView()
	}
	if row.Clicked() {
		m.Choose(i)
	}
	return row
}

// Choose makes the edits of the action i of Items and closes the menu;
// Close closes it without making one. Both happen once the menu is
// built.
func (m *ActionMenu) Choose(i int) {
	if i >= 0 && i < len(m.ed.actions.actions) {
		m.chosen = i
	}
}

func (m *ActionMenu) Close() { m.closing = true }

// Font is the editor's font, LineHeight the height of its lines, and
// Theme its colors, to show the actions as the completions show theirs.
func (m *ActionMenu) Font() ui.Font       { return m.ed.view.font }
func (m *ActionMenu) LineHeight() float32 { return m.ed.view.lineHeight }
func (m *ActionMenu) Theme() *Theme       { return m.ed.theme }

// DefaultCodeActions builds the editor's own menu of what can be done
// into panel: the title of each action, nine rows high.
func DefaultCodeActions(c *ui.Context, panel ui.Element, m *ActionMenu) {
	t := c.Theme()
	font := m.Font()
	lh := m.LineHeight()
	panel.Background(t.Surface).Border(1, t.Border).Radius(t.Radius).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.2)).
		Padding(t.Space(1)).MaxHeight(lh*9 + t.Space(2)).MinWidth(t.Space(40)).MaxWidth(t.Space(150))
	items := m.Items()
	panel.Children(func() {
		ui.Scroll(c).Key("rows").MaxHeight(lh * 9).Children(func() {
			for i := range items {
				if i >= maxListRows {
					break
				}
				row := m.Item(i).Row().AlignItems(ui.Center).Padding(t.Space(0.5), t.Space(1.5)).
					Radius(t.Radius - 2).Height(lh + 2)
				if i == m.Selected() {
					row.Background(t.Accent.Alpha(0.18))
				}
				row.Children(func() {
					ui.Text(c, m.Title(i)).Font(font.Family).FontSize(font.Size).SingleLine().Shrink(1)
				})
			}
		})
	})
}

// buildCodeActions places the menu of what can be done at the caret,
// above the window's content, and makes the edits of the one chosen.
func (ed *editor) buildCodeActions(c *ui.Context, anchor ui.Element) {
	a := &ed.actions
	ui.Overlay(c, func() {
		panel := ui.Box(c).Key("code actions").AttachTo(anchor, ui.AnchorBottomLeft, ui.AnchorTopLeft).
			Role(ui.RoleList).Label(l10n.In(c.Locale()).CodeActions()).Cursor(ui.CursorDefault).KeepFocus().HandleInput(shield)
		render := ed.o.renderActions
		if render == nil {
			render = DefaultCodeActions
		}
		m := &ActionMenu{c: c, ed: ed, chosen: -1}
		render(c, panel, m)
		switch {
		case m.chosen >= 0 && m.chosen < len(a.actions):
			act := a.actions[m.chosen]
			a.close()
			ed.applyEdits(act.Edits)
			c.Invalidate()
		case m.closing:
			a.close()
			c.Invalidate()
		}
	})
	a.scrollTo = false
}

// RenameBar is an editor's bar renaming a name, as RenderRename builds
// it: the new name typed over the old, which OnRename is asked about as
// the bar is submitted.
type RenameBar struct {
	c  *ui.Context
	ed *editor
}

// Text is the text of the bar's field, to bind an input to, with Input.
func (b *RenameBar) Text() *string { return &b.ed.rename.text }

// OldName is the name being renamed, and Range where it is in the text.
func (b *RenameBar) OldName() string { return b.ed.rename.oldName }

func (b *RenameBar) Range() (start, end int) { return b.ed.rename.start, b.ed.rename.end }

// Input makes in the bar's field: it takes the focus with the old name
// selected as the bar opens, Enter asks for the name typed, and Escape
// closes the bar. It returns in.
func (b *RenameBar) Input(in ui.Element) ui.Element {
	r := &b.ed.rename
	if r.focus {
		r.focus = false
		in.Focus().SetTextSelection(0, utf8.RuneCountInString(r.text))
	}
	switch {
	case in.Submitted():
		r.submit(b.ed)
	case in.Shortcut(0, ui.KeyEscape):
		b.Close()
	}
	return in
}

func (b *RenameBar) Close() { b.ed.rename.close(b.ed) }

// DefaultRename builds the editor's own rename bar into bar: the field
// of the new name.
func DefaultRename(c *ui.Context, bar ui.Element, b *RenameBar) {
	t := c.Theme()
	tx := l10n.In(c.Locale())
	bar.Row().Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(1.5), t.Space(2)).Background(t.Surface).
		Border(1, t.Border).Radius(t.Radius).Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.18))
	bar.Children(func() {
		b.Input(ui.TextInputBase(c, b.Text()).Width(t.Space(40)).Padding(t.Space(1), t.Space(2)).Background(t.Background).
			Border(1, t.Border).Radius(t.Radius - 2).Label(tx.Rename()).Placeholder(tx.RenameTo()))
	})
}

// buildRename places the rename bar at the editor's top, above the
// window's content.
func (ed *editor) buildRename(c *ui.Context) {
	t := c.Theme()
	anchor := ui.Box(c).Key("code rename anchor").Absolute().Top(t.Space(1)).Left(0).Right(0).Height(0).PassThrough()
	ui.Overlay(c, func() {
		bar := ui.Box(c).Key("code rename").AttachTo(anchor, ui.AnchorBottom, ui.AnchorTop).
			Role(ui.RoleDialog).Label(l10n.In(c.Locale()).Rename()).Cursor(ui.CursorDefault).HandleInput(shield)
		render := ed.o.renderRename
		if render == nil {
			render = DefaultRename
		}
		render(c, bar, &RenameBar{c: c, ed: ed})
	})
}

// shield takes no input, but keeps the presses on what an editor shows
// over its text, as its find bar, from the editor around it: the input
// goes to the innermost element taking it as it comes, which then leaves
// it to the buttons and fields inside, as they take it.
func shield(ui.InputEvent) bool { return false }

// wheelOnly shields the presses on what a diff shows over its editors,
// outside them, as its separators, but scrolls the editor by the wheel:
// the input never reaches an editor from what is not inside it.
func (ed *editor) wheelOnly(ev ui.InputEvent) bool {
	return ev.Kind == ui.InputScroll && ed.wheel(ev)
}
