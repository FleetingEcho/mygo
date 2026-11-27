package editor

import (
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// LineAnnotation is application text shown in a column to the left of the
// line numbers, as a source line's commit summary and date. Line is
// zero-based. A fixed-width Date comes first, followed by Avatar and Text.
// Detail is kept at the column's right edge while Text is ellipsized to the
// room left for it. Empty fields reserve no extra space.
//
// SetLineAnnotations clears annotations when the buffer changes. Reapply them
// once the new text has been attributed; stale annotations never follow edits
// onto source lines they do not describe.
type LineAnnotation struct {
	Line   int
	Date   string
	Text   string
	Detail string
	Click  func()
	Avatar *LineAnnotationAvatar
	// LastLine is the inclusive end of this annotation's range. Border runs
	// continuously at the range's left edge while its summary stays on Line.
	LastLine int
	Border   ui.Color
	// ActionClick shows a stacked-file history action at the column's trailing
	// edge, as "view blame prior to this change". Its hit area is separate.
	ActionClick func()
	// DetailsClick shows an information button beside the history action, for
	// the app to open the range's commit details. Hovering the column never
	// opens anything; details open only from this button.
	DetailsClick func()
	// ActionTooltip and DetailsTooltip name the two buttons when the pointer
	// rests on them. Without one, the button shows no tooltip.
	ActionTooltip  string
	DetailsTooltip string
}

// lineAnnotationActionSlot is one trailing action button.
type lineAnnotationActionSlot struct {
	click func()
	tip   string
	paint func(p *ui.Painter, x, y float32, color, background ui.Color)
}

// actionSlots lists the trailing buttons from the column's right edge inward:
// history, then details.
func (a *LineAnnotation) actionSlots() []lineAnnotationActionSlot {
	var slots []lineAnnotationActionSlot
	if a.ActionClick != nil {
		slots = append(slots, lineAnnotationActionSlot{a.ActionClick, a.ActionTooltip, paintLineAnnotationAction})
	}
	if a.DetailsClick != nil {
		slots = append(slots, lineAnnotationActionSlot{a.DetailsClick, a.DetailsTooltip, paintLineAnnotationDetails})
	}
	return slots
}

// lineAnnotationResize keeps pointer geometry out of the app model. The new
// width is published through State.OnLineAnnotationResize.
type lineAnnotationResize struct {
	active             bool
	startX, startWidth float32
}

// LineAnnotationAvatar is a small author image at the start of an annotation.
// Bitmap is an optional non-blocking cache lookup: nil keeps the generated
// initial, allowing an image to appear on a later frame without rebuilding the
// annotation. Background is the initial avatar's stable author color.
type LineAnnotationAvatar struct {
	Initial    string
	Background ui.Color
	Bitmap     func() *ui.Bitmap
}

// The trailing action has 6 DIPs leading and 12 DIPs trailing clearance
// around its 16-DIP glyph.
const lineAnnotationActionWidth = float32(34)

// SetLineAnnotations shows annotations to the left of the line numbers.
// Entries may arrive in any order; the last one for a line wins.
func (st *State) SetLineAnnotations(annotations []LineAnnotation) {
	st.ensure()
	next := make([]LineAnnotation, 0, len(annotations))
	for _, annotation := range annotations {
		if annotation.Line < 0 || annotation.Line >= st.ed.view.text.LineCount() {
			continue
		}
		annotation.Date = singleLineAnnotation(annotation.Date)
		annotation.Text = singleLineAnnotation(annotation.Text)
		annotation.Detail = singleLineAnnotation(annotation.Detail)
		if annotation.LastLine < annotation.Line {
			annotation.LastLine = annotation.Line
		}
		annotation.LastLine = min(annotation.LastLine, st.ed.view.text.LineCount()-1)
		next = append(next, annotation)
	}
	slices.SortStableFunc(next, func(a, b LineAnnotation) int { return a.Line - b.Line })
	out := next[:0]
	for _, annotation := range next {
		if len(out) > 0 && out[len(out)-1].Line == annotation.Line {
			out[len(out)-1] = annotation
			continue
		}
		out = append(out, annotation)
	}
	st.ed.lineAnnotations = out
	st.ed.needsBuild = true
}

// LineAnnotations returns the annotations currently shown, in line order.
func (st *State) LineAnnotations() []LineAnnotation {
	st.ensure()
	return slices.Clone(st.ed.lineAnnotations)
}

func singleLineAnnotation(text string) string {
	if !strings.ContainsAny(text, "\r\n") {
		return text
	}
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return ' '
		}
		return r
	}, text)
}

// lineAnnotationFor returns the annotation attached to line.
func (ed *editor) lineAnnotationFor(line int) *LineAnnotation {
	index := sort.Search(len(ed.lineAnnotations), func(i int) bool {
		return ed.lineAnnotations[i].Line >= line
	})
	if index >= len(ed.lineAnnotations) || ed.lineAnnotations[index].Line != line {
		return nil
	}
	return &ed.lineAnnotations[index]
}

// lineAnnotationRange returns the range that decorates line, without putting
// its summary, avatar or actions on continuation lines.
func (ed *editor) lineAnnotationRange(line int) *LineAnnotation {
	index := sort.Search(len(ed.lineAnnotations), func(i int) bool {
		return ed.lineAnnotations[i].Line > line
	}) - 1
	if index < 0 || line > ed.lineAnnotations[index].LastLine {
		return nil
	}
	return &ed.lineAnnotations[index]
}

// lineAnnotationAt returns the annotation in the extra gutter column at x, y.
func (ed *editor) lineAnnotationAt(x, y float32) *LineAnnotation {
	if ed.annotationW <= 0 || x < 0 || x >= ed.annotationW {
		return nil
	}
	_, cy := ed.toContent(x, y)
	line := ed.view.heights.lineAt(cy)
	if line < 0 || line >= ed.view.text.LineCount() {
		return nil
	}
	geometry := ed.view.heights.geometry(line)
	if cy < geometry.rows || cy >= geometry.below {
		return nil
	}
	return ed.lineAnnotationFor(line)
}

// lineAnnotationSlotAt returns the annotation and trailing action button that
// contain x, y. The remaining annotation area keeps its ordinary Click.
func (ed *editor) lineAnnotationSlotAt(x, y float32) (*LineAnnotation, *lineAnnotationActionSlot) {
	annotation := ed.lineAnnotationAt(x, y)
	if annotation == nil {
		return nil, nil
	}
	slots := annotation.actionSlots()
	index := int((ed.annotationW - x) / lineAnnotationActionWidth)
	if x >= ed.annotationW || index < 0 || index >= len(slots) {
		return annotation, nil
	}
	return annotation, &slots[index]
}

// lineAnnotationActionAt returns the click handler of the trailing action icon
// that contains x, y.
func (ed *editor) lineAnnotationActionAt(x, y float32) func() {
	if _, slot := ed.lineAnnotationSlotAt(x, y); slot != nil {
		return slot.click
	}
	return nil
}

// lineAnnotationResizeAt reports whether x is in the trailing separator's
// six-DIP hit area. Three DIPs on either side make the one-DIP line easy to drag.
func (ed *editor) lineAnnotationResizeAt(x float32) bool {
	return len(ed.lineAnnotations) > 0 && ed.annotationW > 0 &&
		x >= ed.annotationW-3 && x <= ed.annotationW+3
}

func (ed *editor) dragLineAnnotation(x float32) {
	drag := ed.annotationResize
	if !drag.active {
		return
	}
	available := max(float32(1), ed.laidW*0.55-(ed.gutterW-ed.annotationW))
	minimum := min(float32(160), available)
	width := min(available, max(minimum, drag.startWidth+x-drag.startX))
	ed.annotationW = width
	ed.o.lineAnnotationWidth = width
	if ed.o.onLineAnnotationResize != nil {
		ed.o.onLineAnnotationResize(width)
	}
}

// fitLineAnnotationText shortens text to width without changing the app's
// original annotation. Shaping the candidates keeps CJK and proportional fonts
// within the column too.
func (v *view) fitLineAnnotationText(text string, width float32) string {
	if text == "" || width <= 0 {
		return ""
	}
	if v.shaped.get(text, v.style).Width <= width {
		return text
	}
	const ellipsis = "…"
	if v.shaped.get(ellipsis, v.style).Width > width {
		return ""
	}
	starts := make([]int, 0, len(text)+1)
	for index := range text {
		starts = append(starts, index)
	}
	starts = append(starts, len(text))
	lo, hi, best := 0, len(starts)-1, 0
	for lo <= hi {
		mid := int(uint(lo+hi) >> 1)
		candidate := text[:starts[mid]] + ellipsis
		if v.shaped.get(candidate, v.style).Width <= width {
			best = starts[mid]
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return text[:best] + ellipsis
}

// lineAnnotationDateWidth reserves a stable date column, keeping avatars
// aligned when one has a one-digit day and another has a two-digit day. The
// extra space after the comma accommodates differences between month glyphs.
func (v *view) lineAnnotationDateWidth() float32 {
	return v.shaped.get("Sep 22, 2000", v.style).Width
}

func (v *view) paintLineAnnotationText(p *ui.Painter, text string, x, y float32, color ui.Color) {
	if text == "" {
		return
	}
	layout := v.shaped.get(text, v.style)
	line := lineLayout{disp: text}
	row := rowLayout{to: len(text), pieces: []rowPiece{{
		to: len(text), l: layout, ascii: isASCII(text), width: layout.Width,
	}}}
	var colors runColors
	v.paintRow(p, &line, &row, x, y, &colors, func(int, ui.Color) ui.Color { return color })
}

// paintLineAnnotation draws one source line's annotation at y.
func (ed *editor) paintLineAnnotation(p *ui.Painter, r ui.Rect, line int, y, height float32, active bool) {
	annotation := ed.lineAnnotationFor(line)
	if annotation == nil || ed.annotationW <= 0 {
		return
	}
	const horizontalPadding = float32(8)
	const detailGap = float32(10)
	const dateGap = float32(12)
	x := r.X + horizontalPadding
	width := ed.annotationW - horizontalPadding*2
	slots := annotation.actionSlots()
	width -= lineAnnotationActionWidth * float32(len(slots))
	date := ""
	if annotation.Date != "" {
		dateWidth := ed.view.lineAnnotationDateWidth()
		date = ed.view.fitLineAnnotationText(annotation.Date, dateWidth)
		x += dateWidth + dateGap
		width -= dateWidth + dateGap
	}
	if avatar := annotation.Avatar; avatar != nil {
		size := min(float32(18), ed.view.lineHeight*0.82)
		rect := ui.Rect{X: x, Y: y + (ed.view.lineHeight-size)/2, W: size, H: size}
		ed.paintLineAnnotationAvatar(p, rect, avatar)
		x += size + 6
		width -= size + 6
	}
	if active {
		p.Fill(ui.Rect{X: r.X, Y: y, W: ed.annotationW, H: height}, ed.theme.ActiveLine, 0)
	}
	color := ed.theme.LineNumber
	if active {
		color = ed.theme.ActiveLineNumber
	}
	if date != "" {
		ed.view.paintLineAnnotationText(p, date, r.X+horizontalPadding, y, color)
	}
	detail := ""
	if annotation.Detail != "" {
		detail = ed.view.fitLineAnnotationText(annotation.Detail, width*0.45)
	}
	detailWidth := float32(0)
	if detail != "" {
		detailWidth = ed.view.shaped.get(detail, ed.view.style).Width
	}
	textWidth := width
	if detail != "" {
		textWidth = max(0, width-detailWidth-detailGap)
	}
	text := ed.view.fitLineAnnotationText(annotation.Text, textWidth)
	ed.view.paintLineAnnotationText(p, text, x, y, color)
	if detail != "" {
		ed.view.paintLineAnnotationText(p, detail, x+width-detailWidth, y, color)
	}
	for index, slot := range slots {
		x := r.X + ed.annotationW - lineAnnotationActionWidth*float32(index+1) + 6
		slot.paint(p, x, y+ed.view.lineHeight/2, color, ed.theme.Background)
	}
}

func (ed *editor) paintLineAnnotationAvatar(p *ui.Painter, rect ui.Rect, avatar *LineAnnotationAvatar) {
	bitmap := (*ui.Bitmap)(nil)
	if avatar.Bitmap != nil {
		bitmap = avatar.Bitmap()
	}
	if bitmap != nil {
		p.Clip(rect, rect.W/2, func() { p.Image(bitmap, rect, ui.Cover) })
		return
	}
	background := avatar.Background
	if background == (ui.Color{}) {
		background = ed.theme.LineNumber
	}
	p.Fill(rect, background, rect.W/2)
	if avatar.Initial == "" {
		return
	}
	style := ed.view.style
	style.Size = max(float32(8), rect.H*0.58)
	layout := ed.view.shaped.get(avatar.Initial, style)
	if len(layout.Lines) == 0 {
		return
	}
	line := layout.Lines[0]
	baseline := rect.Y + (rect.H-(line.Ascent+line.Descent))/2 + line.Ascent
	p.TextLine(layout, rect.X+(rect.W-layout.Width)/2, baseline, func(int) ui.Color {
		return ui.RGB(255, 255, 255)
	})
}

// lineAnnotationGlyphScale shrinks the trailing glyphs below their 16-DIP slot
// so they sit quietly beside the text instead of competing with it.
const lineAnnotationGlyphScale = float32(0.72)

// paintLineAnnotationDetails draws a circled "i": the commit's details.
func paintLineAnnotationDetails(p *ui.Painter, x, y float32, color, _ ui.Color) {
	const k = lineAnnotationGlyphScale
	cx := x + 8
	ring := ui.Path{}
	ring.Circle(cx, y, 7*k)
	p.StrokePath(&ring, 1, color)
	dot := ui.Path{}
	dot.Circle(cx, y-3*k, 0.8)
	p.FillPath(&dot, color)
	stem := ui.Path{}
	stem.MoveTo(cx, y-0.9*k).LineTo(cx, y+3.6*k)
	p.StrokePath(&stem, 1, color)
}

// paintLineAnnotationAction draws three stacked documents with a folded
// corner on the front page: GitLab's history affordance, not a copy action.
func paintLineAnnotationAction(p *ui.Painter, x, y float32, color, background ui.Color) {
	const k = lineAnnotationGlyphScale
	// Offsets are authored on the 16-DIP grid and scaled about its center.
	at := func(ox, oy float32) (float32, float32) { return x + 8 + (ox-8)*k, y + oy*k }
	document := func(ox, oy, w, h float32) {
		left, top := at(ox, oy)
		rect := ui.Rect{X: left, Y: top, W: w * k, H: h * k}
		p.Fill(rect, background, 0)
		path := ui.Path{}
		path.MoveTo(rect.X, rect.Y).LineTo(rect.X+rect.W, rect.Y).LineTo(rect.X+rect.W, rect.Y+rect.H).LineTo(rect.X, rect.Y+rect.H).Close()
		p.StrokePath(&path, 1, color)
	}
	document(0, -4, 6, 10)
	document(3, -5, 6, 11)

	left, top := at(6, -7)
	right, bottom := at(16, 7)
	fold := 3 * k
	front := ui.Path{}
	front.MoveTo(left, top).LineTo(right-fold, top).LineTo(right, top+fold).
		LineTo(right, bottom).LineTo(left, bottom).Close()
	p.FillPath(&front, background)
	p.StrokePath(&front, 1, color)
	crease := ui.Path{}
	crease.MoveTo(right-fold, top).LineTo(right-fold, top+fold).LineTo(right, top+fold)
	p.StrokePath(&crease, 1, color)
}

// paintLineAnnotations draws only rows in view. The column never horizontally
// scrolls, and its separator stays between the annotations and line numbers.
func (ed *editor) paintLineAnnotations(p *ui.Painter, r ui.Rect, f *frame) {
	if ed.annotationW <= 0 {
		return
	}
	v := &ed.view
	h := &v.heights
	column := ui.Rect{X: r.X, Y: r.Y, W: ed.annotationW, H: r.H}
	p.Clip(column, 0, func() {
		for line := f.first; line <= f.last; line++ {
			geometry := h.geometry(line)
			if geometry.hidden {
				continue
			}
			y := r.Y + float32(geometry.rows-ed.scrollY) + ed.sticky
			height := float32(geometry.below - geometry.rows)
			if annotation := ed.lineAnnotationRange(line); annotation != nil && annotation.Border.A > 0 {
				// A pixel is left clear at each end of a range, so two ranges
				// that touch read as two bars, as GitLab's blame does.
				top, bottom := y, y+height
				if line == annotation.Line {
					top += lineAnnotationBorderGap
				}
				if line == annotation.LastLine {
					bottom -= lineAnnotationBorderGap
				}
				if bottom > top {
					p.Fill(ui.Rect{X: r.X + 1, Y: top, W: 3, H: bottom - top}, annotation.Border, 0)
				}
			}
			active := ed.hoverLine == line || f.activeLines[line]
			ed.paintLineAnnotation(p, r, line, y, height, active)
		}
		p.Fill(ui.Rect{X: r.X + ed.annotationW - 1, Y: r.Y, W: 1, H: r.H}, ed.theme.StickyBorder, 0)
	})
}

// lineAnnotationBorderGap is the clear space, in DIPs, at each end of a range's
// border: two together separate neighbouring ranges.
const lineAnnotationBorderGap = float32(1)

// annotationTipDelay is how long the pointer rests on a button before its name shows.
const annotationTipDelay = 350 * time.Millisecond

// paintAnnotationTip names the annotation button under the resting pointer in a
// small card just above it, or below it when the row is at the top. It is drawn
// by the editor, over everything it paints, so it stays beside the button
// instead of anchoring to the text.
func (ed *editor) paintAnnotationTip(p *ui.Painter, r ui.Rect) {
	if ed.annotationTip == "" || !ed.pointerIn {
		return
	}
	if p.Now().Sub(ed.annotationTipSince) < annotationTipDelay {
		p.AnimationFrame()
		return
	}
	v := &ed.view
	const padX, padY, gap = float32(9), float32(5), float32(4)
	layout := v.shaped.get(ed.annotationTip, v.style)
	w, h := layout.Width+padX*2, v.lineHeight+padY*2
	x := min(max(r.X+ed.annotationTipX-w/2, r.X+gap), r.X+r.W-w-gap)
	y := r.Y + ed.annotationTipRow - h - gap
	if y < r.Y+gap {
		y = r.Y + ed.annotationTipRow + v.lineHeight + gap
	}
	card := ui.Rect{X: x, Y: y, W: w, H: h}
	p.Fill(card, ed.theme.Background, 5)
	p.Stroke(card, ed.theme.StickyBorder, 5, 1)
	v.paintLineAnnotationText(p, ed.annotationTip, x+padX, y+padY, ed.theme.Text)
}
