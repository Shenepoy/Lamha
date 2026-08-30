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
	// GTK4 equivalent of AdwStyleManager color-scheme:
	// system → prefer-light unless the desktop prefers dark
	// light  → force-light
	// dark   → force-dark
	// https://gnome.pages.gitlab.gnome.org/libadwaita/doc/main/styles-and-appearance.html
	switch prefs.NormalizeTheme(theme) {
	case prefs.ThemeDark:
		settings.SetObjectProperty("gtk-application-prefer-dark-theme", true)
	case prefs.ThemeLight:
		settings.SetObjectProperty("gtk-application-prefer-dark-theme", false)
	default:
		settings.SetObjectProperty("gtk-application-prefer-dark-theme", desktopPrefersDark())
	}
}

func styleDim(w gtk.Widgetter) {
	if w == nil {
		return
	}
	widget := gtk.BaseWidget(w)
	widget.AddCSSClass("dim-label")
	widget.AddCSSClass("dimmed")
}

func (w *Window) setTheme(theme string) {
	applyTheme(theme)
	if w != nil && w.editor != nil {
		w.editor.refreshChromeTheme()
	}
}

func themePrefersDark() bool {
	switch prefs.NormalizeTheme(prefs.Current().Theme()) {
	case prefs.ThemeDark:
		return true
	case prefs.ThemeLight:
		return false
	default:
		return desktopPrefersDark()
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

func desktopAccentRGB() (r, g, b float64) {
	// Adwaita accent backgrounds: https://gnome.pages.gitlab.gnome.org/libadwaita/doc/main/css-variables.html
	name := "blue"
	source := gio.SettingsSchemaSourceGetDefault()
	if source != nil && source.Lookup("org.gnome.desktop.interface", true) != nil {
		name = gio.NewSettings("org.gnome.desktop.interface").String("accent-color")
	}
	switch strings.ToLower(name) {
	case "teal":
		return 0.129, 0.565, 0.643
	case "green":
		return 0.227, 0.580, 0.290
	case "yellow":
		return 0.784, 0.533, 0
	case "orange":
		return 0.929, 0.357, 0
	case "red":
		return 0.902, 0.176, 0.259
	case "pink":
		return 0.835, 0.380, 0.600
	case "purple":
		return 0.569, 0.255, 0.675
	case "slate":
		return 0.435, 0.514, 0.588
	default:
		return 0.208, 0.518, 0.894
	}
}
