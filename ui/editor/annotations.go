package editor

import (
	"slices"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
)

// LineAnnotation is application text shown in a column to the left of the
// line numbers, as a source line's commit summary and relative date. Line is
// zero-based. Detail is kept at the column's right edge while Text is
// ellipsized to the room left for it. An empty Text and Detail reserves only
// the line for hover or gutter selection.
//
// SetLineAnnotations clears annotations when the buffer changes. Reapply them
// once the new text has been attributed; stale annotations never follow edits
// onto source lines they do not describe.
type LineAnnotation struct {
	Line   int
	Text   string
	Detail string
	Click  func()
	Avatar *LineAnnotationAvatar
	// ActionClick shows a two-page action at the column's trailing edge, as
	// "view blame prior to this change". Its hit area is separate from Click.
	ActionClick func()
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

const lineAnnotationActionWidth = float32(26)

// SetLineAnnotations shows annotations to the left of the line numbers.
// Entries may arrive in any order; the last one for a line wins.
func (st *State) SetLineAnnotations(annotations []LineAnnotation) {
	st.ensure()
	next := make([]LineAnnotation, 0, len(annotations))
	for _, annotation := range annotations {
		if annotation.Line < 0 || annotation.Line >= st.ed.view.text.LineCount() {
			continue
		}
		annotation.Text = singleLineAnnotation(annotation.Text)
		annotation.Detail = singleLineAnnotation(annotation.Detail)
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

// lineAnnotationActionAt returns the annotation whose trailing action icon
// contains x, y. The remaining annotation area keeps its ordinary Click.
func (ed *editor) lineAnnotationActionAt(x, y float32) *LineAnnotation {
	annotation := ed.lineAnnotationAt(x, y)
	if annotation == nil || annotation.ActionClick == nil || x < ed.annotationW-lineAnnotationActionWidth {
		return nil
	}
	return annotation
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
	x := r.X + horizontalPadding
	width := ed.annotationW - horizontalPadding*2
	if annotation.ActionClick != nil {
		width -= lineAnnotationActionWidth
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
	if annotation.ActionClick != nil {
		paintLineAnnotationAction(p, r.X+ed.annotationW-lineAnnotationActionWidth+6, y+ed.view.lineHeight/2, color, ed.theme.Background)
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

// paintLineAnnotationAction draws two overlapping pages without depending on
// an icon font, so every theme and platform gets the same recognizable action.
func paintLineAnnotationAction(p *ui.Painter, x, y float32, color, background ui.Color) {
	back := ui.Path{}
	back.MoveTo(x+1, y-6).LineTo(x+9, y-6).LineTo(x+9, y+3).LineTo(x+3, y+3).LineTo(x+3, y+5).LineTo(x+1, y+5).Close()
	p.StrokePath(&back, 1.1, color)
	frontRect := ui.Rect{X: x + 5, Y: y - 3, W: 9, H: 10}
	p.Fill(frontRect, background, 0)
	front := ui.Path{}
	front.MoveTo(x+5, y-3).LineTo(x+14, y-3).LineTo(x+14, y+7).LineTo(x+5, y+7).Close()
	p.StrokePath(&front, 1.1, color)
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
			active := ed.hoverLine == line || f.activeLines[line]
			ed.paintLineAnnotation(p, r, line, y, height, active)
		}
		p.Fill(ui.Rect{X: r.X + ed.annotationW - 1, Y: r.Y, W: 1, H: r.H}, ed.theme.StickyBorder, 0)
	})
}
