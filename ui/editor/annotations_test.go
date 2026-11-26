package editor

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestLineAnnotationsShareGutterAndClickWithoutMovingCaret(t *testing.T) {
	ct := newEditorTest(t, "", "|first\nsecond\nthird\n")
	ct.st.LineAnnotationWidth = 240
	clicked := 0
	ct.st.SetLineAnnotations([]LineAnnotation{
		{Line: 2, Text: "Third change"},
		{Line: 1, Text: "Ada • Explain the line annotation model", Detail: "3 weeks ago", Click: func() { clicked++ }},
	})
	ct.tt.Frame()
	ed := &ct.st.ed
	if ed.annotationW != 240 || ed.gutterW <= ed.annotationW || ed.textX <= ed.annotationW {
		t.Fatalf("annotation gutter geometry = annotation %v gutter %v text %v", ed.annotationW, ed.gutterW, ed.textX)
	}
	annotations := ct.st.LineAnnotations()
	if len(annotations) != 2 || annotations[0].Line != 1 || annotations[1].Line != 2 {
		t.Fatalf("annotations were not normalized by line: %#v", annotations)
	}
	if annotation := ed.lineAnnotationAt(ed.annotationW/2, ed.view.lineHeight*1.5); annotation == nil || annotation.Line != 1 {
		t.Fatalf("annotation hit test = %#v, want line 1", annotation)
	}

	ct.tt.Press(ed.annotationW/2, ed.view.lineHeight*1.5)
	ct.tt.Release(ed.annotationW/2, ed.view.lineHeight*1.5)
	if clicked != 1 {
		t.Fatalf("annotation click ran %d times, want 1", clicked)
	}
	if got := ct.st.CursorPos(); got.Line != 0 || got.Col != 0 {
		t.Fatalf("annotation click moved the caret to %+v", got)
	}
}

func TestLineAnnotationActionUsesSeparateClickArea(t *testing.T) {
	ct := newEditorTest(t, "", "|first\nsecond\nthird\n")
	ct.st.LineAnnotationWidth = 240
	summaryClicks, actionClicks := 0, 0
	ct.st.SetLineAnnotations([]LineAnnotation{{
		Line: 1, Text: "Ada • second change", Detail: "yesterday",
		Click:       func() { summaryClicks++ },
		ActionClick: func() { actionClicks++ },
	}})
	ct.tt.Frame()
	ed := &ct.st.ed
	y := ed.view.lineHeight * 1.5
	ct.tt.Press(ed.annotationW-lineAnnotationActionWidth/2, y)
	ct.tt.Release(ed.annotationW-lineAnnotationActionWidth/2, y)
	if actionClicks != 1 || summaryClicks != 0 {
		t.Fatalf("action clicks = %d, summary clicks = %d", actionClicks, summaryClicks)
	}
	x := ed.annotationW - lineAnnotationActionWidth - 10
	ct.tt.Press(x, y)
	ct.tt.Release(x, y)
	if summaryClicks != 1 || actionClicks != 1 {
		t.Fatalf("summary clicks = %d, action clicks = %d", summaryClicks, actionClicks)
	}
	if got := ct.st.CursorPos(); got.Line != 0 || got.Col != 0 {
		t.Fatalf("annotation clicks moved the caret to %+v", got)
	}
}

func TestLineAnnotationAvatarPaintsGeneratedFallback(t *testing.T) {
	ct := newEditorTest(t, "", "|first\n")
	ct.st.LineAnnotationWidth = 240
	lookups := 0
	ct.st.SetLineAnnotations([]LineAnnotation{{
		Line: 0, Text: "Ada • first change",
		Avatar: &LineAnnotationAvatar{
			Initial: "A", Background: ui.RGB(255, 0, 0),
			Bitmap: func() *ui.Bitmap { lookups++; return nil },
		},
	}})
	ct.tt.Frame()
	if lookups == 0 {
		t.Fatal("the avatar cache was not consulted")
	}
	r, g, b := ct.pixel(11, ct.st.ed.view.lineHeight/2)
	if r < 200 || g > 80 || b > 80 {
		t.Fatalf("avatar fallback pixel = %d,%d,%d, want the author's red circle", r, g, b)
	}
}

func TestLineAnnotationHoverRequestsItsSourceLineEnd(t *testing.T) {
	ct := newEditorTest(t, "", "first\nsecond\nthird\n")
	ct.st.LineAnnotationWidth = 260
	ct.st.HoverDelay = 100 * time.Millisecond
	var requests []HoverRequest
	ct.st.OnHover = func(request HoverRequest) { requests = append(requests, request) }
	ct.st.SetLineAnnotations([]LineAnnotation{{Line: 1, Text: "Ada • second change", Detail: "yesterday"}})
	ct.tt.Frame()
	ed := &ct.st.ed

	ct.tt.Move(ed.annotationW/2, ed.view.lineHeight*1.5)
	ct.tt.Advance(200 * time.Millisecond)
	ct.tt.Frame()
	want := ed.view.text.LineEnd(1)
	if len(requests) != 1 || requests[0].Offset != want || requests[0].Version != ed.view.version {
		t.Fatalf("annotation hover = %#v, want offset %d at version %d", requests, want, ed.view.version)
	}

	ct.tt.Move(ed.annotationW/2, ed.view.lineHeight*2.5)
	ct.tt.Advance(200 * time.Millisecond)
	ct.tt.Frame()
	if len(requests) != 1 {
		t.Fatalf("a line without an annotation requested hover: %#v", requests)
	}
}

func TestLineAnnotationTextElidesWithoutChangingValue(t *testing.T) {
	ct := newEditorTest(t, "", "first\n")
	ct.st.LineAnnotationWidth = 180
	const subject = "A commit subject much too long for the annotation column"
	ct.st.SetLineAnnotations([]LineAnnotation{{Line: 0, Text: subject, Detail: "3 weeks ago"}})
	ct.tt.Frame()
	v := &ct.st.ed.view
	fitted := v.fitLineAnnotationText(subject, 72)
	if fitted == subject || !strings.HasSuffix(fitted, "…") {
		t.Fatalf("long annotation was not ellipsized: %q", fitted)
	}
	if width := v.shaped.get(fitted, v.style).Width; width > 72 {
		t.Fatalf("ellipsized annotation width = %v, want at most 72: %q", width, fitted)
	}
	if got := ct.st.LineAnnotations()[0].Text; got != subject {
		t.Fatalf("layout changed the annotation to %q", got)
	}
	if fitted := v.fitLineAnnotationText("短", 0); fitted != "" {
		t.Fatalf("zero-width annotation = %q", fitted)
	}
}

func TestLineAnnotationsClearWhenTextChanges(t *testing.T) {
	ct := newEditorTest(t, "", "|first\nsecond\n")
	ct.st.LineAnnotationWidth = 240
	ct.st.SetLineAnnotations([]LineAnnotation{{Line: 1, Text: "Second change"}})
	ct.typ("new\n")
	if got := ct.st.LineAnnotations(); len(got) != 0 {
		t.Fatalf("edited text retained stale line annotations: %#v", got)
	}
	ct.st.SetLineAnnotations([]LineAnnotation{{Line: 2, Text: "Recomputed second change"}})
	ct.tt.Frame()
	if annotation := ct.st.ed.lineAnnotationAt(ct.st.ed.annotationW/2, ct.st.ed.view.lineHeight*2.5); annotation == nil || annotation.Line != 2 {
		t.Fatalf("recomputed annotation is not attached to its new line: %#v", annotation)
	}
}

func TestLineAnnotationResizeAndRangeBorder(t *testing.T) {
	ct := newEditorTest(t, "", "|first\nsecond\nthird\nfourth\n")
	ct.st.LineAnnotationWidth = 240
	ct.st.SetLineAnnotations([]LineAnnotation{{
		Line: 0, LastLine: 2, Text: "Ada • a range", Border: ui.RGB(255, 0, 0),
	}})
	resized := float32(0)
	ct.st.OnLineAnnotationResize = func(width float32) {
		resized = width
		ct.st.LineAnnotationWidth = width
	}
	ct.tt.Frame()
	ed := &ct.st.ed
	if annotation := ed.lineAnnotationRange(1); annotation == nil || annotation.Line != 0 || annotation.LastLine != 2 {
		t.Fatalf("continuation has no range decoration: %#v", annotation)
	}
	if ed.lineAnnotationFor(1) != nil || ed.lineAnnotationRange(3) != nil {
		t.Fatal("a range duplicated its summary or overran its last line")
	}
	r, g, b := ct.pixel(2, ed.view.lineHeight*1.5)
	if r < 200 || g > 80 || b > 80 {
		t.Fatalf("continuation border pixel = %d,%d,%d, want red", r, g, b)
	}
	if !ed.lineAnnotationResizeAt(ed.annotationW+3) || ed.lineAnnotationResizeAt(ed.annotationW+4) {
		t.Fatal("the separator hit area is not three DIPs on each side")
	}
	ct.tt.Move(ed.annotationW, 8)
	if got := ct.tt.Cursor(); got != ui.CursorResizeEW {
		t.Fatalf("separator cursor = %v, want horizontal resize", got)
	}
	startX := ed.annotationW
	targetX := startX + 30
	ct.tt.Press(startX, 8)
	ct.tt.Move(targetX, 8)
	ct.tt.Release(targetX, 8)
	if resized != 270 || ct.st.LineAnnotationWidth != 270 {
		t.Fatalf("resized width = %v state = %v, want 270", resized, ct.st.LineAnnotationWidth)
	}
	ct.tt.Frame()
	if ed.annotationW != 270 {
		t.Fatalf("laid-out annotation width = %v, want 270", ed.annotationW)
	}
	ct.tt.Press(ed.annotationW, 8)
	ct.tt.Move(ed.annotationW-200, 8)
	ct.tt.Release(ed.annotationW-200, 8)
	if resized != 160 {
		t.Fatalf("left drag width = %v, want the 160 DIP minimum", resized)
	}
}

func TestLineAnnotationWidthLeavesSourceRoom(t *testing.T) {
	st := &State{Buffer: NewBuffer("first\nsecond\n"), LineAnnotationWidth: 500}
	st.SetLineAnnotations([]LineAnnotation{{Line: 0, Text: "A change"}})
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 300, 100)
	tt.Frame()
	if st.ed.annotationW >= 300*0.55 || st.ed.annotationW <= 0 {
		t.Fatalf("narrow editor annotation width = %v, want within 55%%", st.ed.annotationW)
	}
}
