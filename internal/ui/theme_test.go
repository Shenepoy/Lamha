package ui

import (
	"testing"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/prefs"
)

func TestResolveTheme(t *testing.T) {
	tests := []struct {
		name        string
		theme       string
		desktopDark bool
		wantDark    bool
		wantScheme  gtk.InterfaceColorScheme
	}{
		{"system light", prefs.ThemeSystem, false, false, gtk.InterfaceColorSchemeLight},
		{"system dark", prefs.ThemeSystem, true, true, gtk.InterfaceColorSchemeDark},
		{"forced light", prefs.ThemeLight, true, false, gtk.InterfaceColorSchemeLight},
		{"forced dark", prefs.ThemeDark, false, true, gtk.InterfaceColorSchemeDark},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotDark, gotScheme := resolveTheme(test.theme, test.desktopDark)
			if gotDark != test.wantDark || gotScheme != test.wantScheme {
				t.Fatalf("resolveTheme(%q, %t) = (%t, %v), want (%t, %v)",
					test.theme, test.desktopDark, gotDark, gotScheme, test.wantDark, test.wantScheme)
			}
		})
	}
}
