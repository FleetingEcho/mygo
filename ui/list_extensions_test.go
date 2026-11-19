package ui

import "testing"

func TestListStateChooseChangesSelectedIndex(t *testing.T) {
	selected := 0
	s := ListState{Selected: &selected}
	tt := NewTester(func(c *Context) {
		List(c, &s, 3).Rows(func(row ListRow) {
			Textf(row.Context, "Row %d", row.Index)
		})
	}, 200, 100)
	if err := tt.Click("Row 0"); err != nil {
		t.Fatal(err)
	}
	s.Choose(2)
	tt.Frame()
	if selected != 2 {
		t.Fatalf("Choose selected row %d, want 2", selected)
	}
	if err := tt.Click("Row 1"); err != nil {
		t.Fatal(err)
	}
	if selected != 1 {
		t.Fatalf("ordinary row selection after Choose selected %d, want 1", selected)
	}
}

func TestListEndReachedFollowsUserNavigation(t *testing.T) {
	selected, n := 0, 24
	loaded := []int{}
	s := ListState{Selected: &selected, EndReached: func(last int) {
		loaded = append(loaded, last)
		if n < 36 {
			n += 6
		}
	}}
	tt := NewTester(func(c *Context) {
		List(c, &s, n).Rows(func(row ListRow) {
			Textf(row.Context, "Row %d", row.Index)
		}).Label("Rows").Focus()
	}, 200, 100)
	if len(loaded) != 0 {
		t.Fatalf("programmatic initial layout selected %v and loaded %v", selected, loaded)
	}
	tt.Key(0, KeyEnd)
	if len(loaded) != 1 || loaded[0] != 23 || n != 30 {
		t.Fatalf("End after user navigation loaded %v with %d rows; visible=%d..%d", loaded, n, s.first, s.last)
	}
	tt.Frame()
	if len(loaded) != 1 || n != 30 {
		t.Fatalf("rendering the new rows loaded %v with %d rows", loaded, n)
	}
	n = 40
	s.ScrollToEnd()
	tt.Frame()
	tt.Frame()
	if len(loaded) != 1 {
		t.Fatalf("programmatic ScrollToEnd triggered EndReached: %v", loaded)
	}
	s.ScrollIntoView(0)
	tt.Frame()
	tt.Key(0, KeyEnd)
	if len(loaded) != 2 || loaded[1] != 39 || n != 40 {
		t.Fatalf("second user End loaded %v with %d rows; visible=%d..%d", loaded, n, s.first, s.last)
	}
}

func TestListNearEndTriggersBeforeLastRow(t *testing.T) {
	selected, calls, lastVisible := 0, 0, -1
	var s ListState
	s.Selected, s.NearEnd = &selected, 5
	s.EndReached = func(int) {
		calls++
		lastVisible = s.last
	}
	tt := NewTester(func(c *Context) {
		List(c, &s, 24).Rows(func(row ListRow) {
			Textf(row.Context, "Row %d", row.Index)
		}).Label("Rows").Focus()
	}, 200, 100)
	for range 12 {
		if calls > 0 {
			break
		}
		tt.Key(0, KeyPageDown)
	}
	if calls != 1 {
		t.Fatalf("near-end was reached %d times, selection=%d, visible=%d..%d", calls, selected, s.first, s.last)
	}
	if lastVisible >= 23 {
		t.Fatalf("near-end fired only with the last row visible at %d", lastVisible)
	}
}
