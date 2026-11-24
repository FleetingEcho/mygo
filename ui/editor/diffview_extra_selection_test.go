package editor

import "testing"

func TestExtraSelectedLinesPaintBesideTheMainRange(t *testing.T) {
	d := &DiffState{}
	d.SetSelectedLines(DiffLineRange{Side: DiffNew, Start: 4, End: 4}, true)
	d.SetExtraSelectedLines([]DiffLineRange{
		{Side: DiffNew, Start: 1, End: 2},
		{Side: DiffOld, Start: 7, End: 7},
	})
	for _, test := range []struct {
		side DiffSide
		line int
		want bool
	}{
		{DiffNew, 4, true}, {DiffNew, 1, true}, {DiffNew, 2, true},
		{DiffNew, 3, false}, {DiffNew, 7, false}, {DiffOld, 7, true}, {DiffOld, 4, false},
	} {
		if got := d.lineSelected(test.side, test.line); got != test.want {
			t.Errorf("lineSelected(%v, %d) = %v, want %v", test.side, test.line, got, test.want)
		}
	}
	d.SetExtraSelectedLines(nil)
	if d.lineSelected(DiffNew, 1) {
		t.Error("extra ranges survived being cleared")
	}
}
