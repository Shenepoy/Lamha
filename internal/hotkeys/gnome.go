// Package hotkeys installs system-wide screenshot shortcuts.
package hotkeys

import (
	"fmt"
	"os"
	"strconv"

	"github.com/diamondburned/gotk4/pkg/gio/v2"

	"github.com/lamha-app/lamha/internal/i18n"
)

const (
	mediaKeysSchema = "org.gnome.settings-daemon.plugins.media-keys"
	customSchema    = "org.gnome.settings-daemon.plugins.media-keys.custom-keybinding"
	bindingPrefix   = "/org/gnome/settings-daemon/plugins/media-keys/custom-keybindings/"
)

type binding struct {
	id    string
	name  string
	args  string
	accel string
}

func captureBindings(accels map[string]string) []binding {
	items := []binding{
		{id: "lamha-area", name: i18n.T("Lamha capture area"), args: "--capture=area", accel: "<Control><Shift>A"},
		{id: "lamha-window", name: i18n.T("Lamha capture window"), args: "--capture=window", accel: ""},
		{id: "lamha-screen", name: i18n.T("Lamha capture screen"), args: "--capture=screen", accel: "<Control><Shift>S"},
	}
	for i, item := range items {
		if accels == nil {
			continue
		}
		if accel, ok := accels[item.id]; ok {
			items[i].accel = accel
		}
	}
	return items
}

// InstallGNOME registers capture shortcuts with GNOME Settings Daemon.
func InstallGNOME(accels map[string]string) error {
	if gio.SettingsSchemaSourceGetDefault() == nil {
		return fmt.Errorf("no GSettings schemas available")
	}
	if gio.SettingsSchemaSourceGetDefault().Lookup(mediaKeysSchema, true) == nil {
		return fmt.Errorf("GNOME media-keys schema is not available")
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve lamha binary: %w", err)
	}

	media := gio.NewSettings(mediaKeysSchema)
	existing := media.Strv("custom-keybindings")
	paths := mergeBindingPaths(existing, bindingPaths())
	for _, item := range captureBindings(accels) {
		path := bindingPrefix + item.id + "/"
		custom := gio.NewSettingsWithPath(customSchema, path)
		custom.SetString("name", item.name)
		custom.SetString("command", strconv.Quote(exe)+" "+item.args)
		custom.SetString("binding", item.accel)
	}
	if !sameStrings(existing, paths) {
		if !media.SetStrv("custom-keybindings", paths) {
			return fmt.Errorf("could not update GNOME custom keybindings")
		}
	}
	return nil
}

func bindingPaths() []string {
	items := captureBindings(nil)
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = bindingPrefix + item.id + "/"
	}
	return out
}

func mergeBindingPaths(existing, ours []string) []string {
	out := append([]string{}, existing...)
	seen := make(map[string]bool, len(existing))
	for _, path := range existing {
		seen[path] = true
	}
	for _, path := range ours {
		if !seen[path] {
			out = append(out, path)
			seen[path] = true
		}
	}
	return out
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
