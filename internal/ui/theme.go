package ui

import (
	"strings"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/prefs"
)

func applyTheme(theme string) {
	settings := gtk.SettingsGetDefault()
	if settings == nil {
		return
	}
	switch prefs.NormalizeTheme(theme) {
	case prefs.ThemeDark:
		settings.SetObjectProperty("gtk-application-prefer-dark-theme", true)
	case prefs.ThemeLight:
		settings.SetObjectProperty("gtk-application-prefer-dark-theme", false)
	default:
		settings.SetObjectProperty("gtk-application-prefer-dark-theme", desktopPrefersDark())
	}
}

func desktopPrefersDark() bool {
	source := gio.SettingsSchemaSourceGetDefault()
	if source == nil || source.Lookup("org.gnome.desktop.interface", true) == nil {
		return false
	}
	scheme := gio.NewSettings("org.gnome.desktop.interface").String("color-scheme")
	return strings.Contains(strings.ToLower(scheme), "dark")
}
