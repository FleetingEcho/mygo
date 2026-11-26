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
}

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
