package theme

import "testing"

func TestSetThemeAppliesKnownModes(t *testing.T) {
	var applied []string
	service := New(func(mode string) error {
		applied = append(applied, mode)
		return nil
	})

	for _, mode := range []string{"system", "light", "dark"} {
		if err := service.SetTheme(mode); err != nil {
			t.Fatalf("SetTheme(%q): %v", mode, err)
		}
	}
	if len(applied) != 3 || applied[0] != "system" || applied[1] != "light" || applied[2] != "dark" {
		t.Fatalf("applied = %v", applied)
	}
}

func TestSetThemeRejectsUnknownMode(t *testing.T) {
	service := New(func(string) error {
		t.Fatal("invalid mode must not be applied")
		return nil
	})

	if err := service.SetTheme("sepia"); err == nil {
		t.Fatal("SetTheme accepted an unknown mode")
	}
}
