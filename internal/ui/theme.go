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
	preferDark, colorScheme := resolveTheme(theme, desktopPrefersDark())

	// GTK 4.20+ themes use prefers-color-scheme media queries. The old
	// application preference only selects a separate gtk-dark.css variant, so
	// it does not switch the GTK 4.22 default theme bundled in the AppImage.
	// Keep both properties in sync for modern and traditional GTK themes.
	settings.SetObjectProperty("gtk-application-prefer-dark-theme", preferDark)
	settings.SetObjectProperty("gtk-interface-color-scheme", colorScheme)
}

func resolveTheme(theme string, desktopDark bool) (bool, gtk.InterfaceColorScheme) {
	dark := desktopDark
	if normalized := prefs.NormalizeTheme(theme); normalized != prefs.ThemeSystem {
		dark = normalized == prefs.ThemeDark
	}
	if dark {
		return true, gtk.InterfaceColorSchemeDark
	}
	return false, gtk.InterfaceColorSchemeLight
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
