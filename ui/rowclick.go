package ui

// RowDoubleClicked reports, while a row of a List or a Table builds, that
// the row was double-clicked since the last frame with the pointer over the
// element. A cell sees its row's double click this way, leaving ordinary
// clicks to the row's selection behavior.
func (e *node) RowDoubleClicked() bool {
	if !e.hasState() {
		return false
	}
	row := e.c.row
	return row != nil && row.e != nil && row.double && e.Hovered()
}

// RowClicked reports, while a row of a List or a Table builds, that the
// row was clicked since the last frame with the pointer over the element:
// a row's content acts on a click this way and leaves the click to the
// row, which goes on choosing, where Clicked would take it, so that the
// row's ListState.Label names what takes the click, for typing its first
// letters, screen readers and Tester.Click alike. It reports each click of
// the primary button without modifiers, on the row chosen too, and not the
// keys that move the choice, nor a Shift- or Cmd-click extending it. A
// double click reports its first click here and its second with
// RowDoubleClicked, as on the systems' own lists. Assistive technology
// pressing the row, without the pointer, reports it to the first element
// asking. In a list that does not choose its rows, asking makes the row
// take the click.
//
//	ui.List(c, &app.files, len(app.names), func(i int) {
//		if ui.Text(c, app.names[i]).Padding(6, 12).FillWidth().RowClicked() {
//			app.open(i)
//		}
//	})
func (e *node) RowClicked() bool {
	if !e.hasState() {
		return false
	}
	row := e.c.row
	if row == nil || row.e == nil {
		return false
	}
	row.e.flags |= flagClickable
	if !row.single {
		return false
	}
	if row.e.Hovered() {
		return e.Hovered()
	}
	// Not by the pointer: no element is under it.
	if row.singleTaken {
		return false
	}
	row.singleTaken = true
	return true
}
