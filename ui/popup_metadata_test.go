package ui

import "testing"

func TestMenuButtonsExposePopupWithoutOverridingExplicitChoice(t *testing.T) {
	tt := NewTester(func(c *Context) {
		ButtonBase(c).Label("Actions").PopoverMenuButton(func(m *Menu) {
			m.Item("Open")
		})
		ButtonBase(c).Label("Choices").HasPopup(PopupDialog).Menu(func(m *Menu) {
			m.Item("One")
		})
		ButtonBase(c).Label("No popup").HasPopup(PopupNone).PopoverMenuButton(func(m *Menu) {
			m.Item("Hidden")
		})
	}, 400, 200)
	for _, test := range []struct {
		name string
		want Popup
	}{
		{"Actions", PopupMenu},
		{"Choices", PopupDialog},
		{"No popup", PopupNone},
	} {
		info, ok := tt.Access(test.name)
		if !ok || info.HasPopup != test.want {
			t.Errorf("%s popup = %v, found=%v; want %v", test.name, info.HasPopup, ok, test.want)
		}
	}
}
