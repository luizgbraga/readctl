package styles

import "testing"

func TestThemeStylesDefined(t *testing.T) {
	// Test that all style variables are defined and not nil
	styles := []struct {
		name  string
		style interface{}
	}{
		{"AppStyle", AppStyle},
		{"Header", Header},
		{"Footer", Footer},
		{"SelectedItem", SelectedItem},
		{"ErrorText", ErrorText},
		{"HelpStyle", HelpStyle},
		{"HelpKeyStyle", HelpKeyStyle},
		{"HelpDescStyle", HelpDescStyle},
		{"ModalStyle", ModalStyle},
		{"ModalTitleStyle", ModalTitleStyle},
	}

	for _, s := range styles {
		if s.style == nil {
			t.Errorf("%s is nil", s.name)
		}
	}
}
