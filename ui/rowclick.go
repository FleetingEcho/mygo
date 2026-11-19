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
// row was clicked with the pointer over this element. Unlike Clicked, it
// leaves the click to the row too, so ordinary row selection still occurs.
// Only primary-button clicks without modifiers qualify. The first click of
// a double click is reported here; its second uses RowDoubleClicked.
//
// If the row was activated by the keyboard or assistive technology, with
// no element under the pointer, the first element asking receives it. In a
// list that does not choose its rows, asking makes the row take the click.
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
	if row.singleTaken {
		return false
	}
	row.singleTaken = true
	return true
}
