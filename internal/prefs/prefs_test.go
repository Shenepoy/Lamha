package prefs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClampZoom(t *testing.T) {
	if ClampZoom(0) != DefaultMagnifierZoom {
		t.Fatalf("ClampZoom(0) = %v", ClampZoom(0))
	}
	if ClampZoom(1) != MinMagnifierZoom {
		t.Fatalf("ClampZoom(1) = %v", ClampZoom(1))
	}
	if ClampZoom(20) != MaxMagnifierZoom {
		t.Fatalf("ClampZoom(20) = %v", ClampZoom(20))
	}
	if ClampZoom(4.5) != 4.5 {
		t.Fatalf("ClampZoom(4.5) = %v", ClampZoom(4.5))
	}
}

func TestNormalizeThemeAndLanguage(t *testing.T) {
	if NormalizeTheme("") != ThemeSystem || NormalizeTheme("dark") != ThemeDark {
		t.Fatal("NormalizeTheme")
	}
	if NormalizeLanguage("") != LanguageSystem || NormalizeLanguage("ar") != LanguageArabic {
		t.Fatal("NormalizeLanguage")
	}
}

func TestSetThemeAndLanguageRoundTrip(t *testing.T) {
	store := &Store{current: defaults(), path: filepath.Join(t.TempDir(), "settings.json")}
	if err := store.SetTheme(ThemeDark); err != nil {
		t.Fatal(err)
	}
	if store.Theme() != ThemeDark {
		t.Fatalf("Theme() = %q", store.Theme())
	}
	if err := store.SetLanguage(LanguageArabic); err != nil {
		t.Fatal(err)
	}
	if store.Language() != LanguageArabic {
		t.Fatalf("Language() = %q", store.Language())
	}
}

func TestSetMagnifierZoomRoundTrip(t *testing.T) {
	store := &Store{current: defaults(), path: filepath.Join(t.TempDir(), "settings.json")}
	if err := store.SetMagnifierZoom(7); err != nil {
		t.Fatal(err)
	}
	if store.MagnifierZoom() != 7 {
		t.Fatalf("MagnifierZoom() = %v", store.MagnifierZoom())
	}
}

func TestToolbarDefaultsAndNormalize(t *testing.T) {
	if NormalizeToolbarEdge("") != ToolbarEdgeTop || NormalizeToolbarEdge("left") != ToolbarEdgeLeft || NormalizeToolbarEdge("Right") != ToolbarEdgeRight || NormalizeToolbarEdge("float") != ToolbarEdgeFloat {
		t.Fatal("NormalizeToolbarEdge")
	}
	if ClampToolbarOffset(-1) != 0 || ClampToolbarOffset(2) != 1 || ClampToolbarOffset(0.25) != 0.25 {
		t.Fatal("ClampToolbarOffset")
	}
	if !ToolbarVertical(ToolbarEdgeLeft) || ToolbarVertical(ToolbarEdgeTop) {
		t.Fatal("ToolbarVertical")
	}
	if ToolbarOffsetForAlign(ToolbarAlignStart) != 0 || ToolbarAlignForOffset(0.9) != ToolbarAlignEnd {
		t.Fatal("toolbar align")
	}
}

func TestSetToolbarPlacementRoundTrip(t *testing.T) {
	store := &Store{current: defaults(), path: filepath.Join(t.TempDir(), "settings.json")}
	if err := store.SetToolbarPlacement(ToolbarEdgeRight, 0.4, 0.2); err != nil {
		t.Fatal(err)
	}
	if store.ToolbarEdge() != ToolbarEdgeRight || store.ToolbarY() != 0.2 || store.ToolbarOffset() != 0.2 {
		t.Fatalf("placement = %s x=%v y=%v", store.ToolbarEdge(), store.ToolbarX(), store.ToolbarY())
	}
	if err := store.SetToolbarLocked(true); err != nil {
		t.Fatal(err)
	}
	if !store.ToolbarLocked() {
		t.Fatal("ToolbarLocked")
	}
	if err := store.SetToolbarSnapAlign(false); err != nil {
		t.Fatal(err)
	}
	if store.ToolbarSnapAlign() {
		t.Fatal("ToolbarSnapAlign")
	}
	if err := store.ResetToolbar(); err != nil {
		t.Fatal(err)
	}
	if store.ToolbarEdge() != DefaultToolbarEdge || store.ToolbarX() != DefaultToolbarOffset || store.ToolbarLocked() || !store.ToolbarSnapAlign() {
		t.Fatal("ResetToolbar")
	}
}

func TestSettingsFromFileToolbarDefaults(t *testing.T) {
	got := settingsFromFile(settingsFile{MagnifierZoom: 4, Theme: ThemeDark, Language: LanguageEnglish})
	if got.ToolbarEdge != DefaultToolbarEdge || got.ToolbarX != DefaultToolbarOffset || !got.ToolbarSnapAlign || got.ToolbarLocked {
		t.Fatalf("missing toolbar fields should keep defaults: %+v", got)
	}
	off := 0.0
	snap := false
	got = settingsFromFile(settingsFile{
		ToolbarEdge:      ToolbarEdgeBottom,
		ToolbarOffset:    &off,
		ToolbarLocked:    true,
		ToolbarSnapAlign: &snap,
	})
	if got.ToolbarEdge != ToolbarEdgeBottom || got.ToolbarX != 0 || !got.ToolbarLocked || got.ToolbarSnapAlign {
		t.Fatalf("explicit toolbar fields: %+v", got)
	}
	rightOff := 0.2
	got = settingsFromFile(settingsFile{ToolbarEdge: ToolbarEdgeRight, ToolbarOffset: &rightOff})
	if got.ToolbarY != 0.2 || got.ToolbarX != DefaultToolbarOffset {
		t.Fatalf("explicit toolbar fields: %+v", got)
	}
	store := &Store{current: defaults(), path: filepath.Join(t.TempDir(), "float.json")}
	if err := store.SetToolbarDock(ToolbarEdgeFloat, 0.3, 0.7, true); err != nil {
		t.Fatal(err)
	}
	if store.ToolbarEdge() != ToolbarEdgeFloat || store.ToolbarX() != 0.3 || store.ToolbarY() != 0.7 || !store.ToolbarStacked() {
		t.Fatalf("float dock = %s %v %v stacked=%v", store.ToolbarEdge(), store.ToolbarX(), store.ToolbarY(), store.ToolbarStacked())
	}
}

func TestStepSettingsDefaultsAndRoundTrip(t *testing.T) {
	got := settingsFromFile(settingsFile{MagnifierZoom: 4, Theme: ThemeDark})
	if !got.RenumberSteps || !got.DeleteZeroSteps {
		t.Fatalf("missing step fields should stay on: %+v", got)
	}
	off := false
	got = settingsFromFile(settingsFile{RenumberSteps: &off, DeleteZeroSteps: &off})
	if got.RenumberSteps || got.DeleteZeroSteps {
		t.Fatalf("explicit step fields: %+v", got)
	}
	store := &Store{current: defaults(), path: filepath.Join(t.TempDir(), "steps.json")}
	if !store.RenumberSteps() || !store.DeleteZeroSteps() {
		t.Fatal("defaults")
	}
	if err := store.SetRenumberSteps(false); err != nil {
		t.Fatal(err)
	}
	if err := store.SetDeleteZeroSteps(false); err != nil {
		t.Fatal(err)
	}
	if store.RenumberSteps() || store.DeleteZeroSteps() {
		t.Fatal("setters")
	}
}

func TestDefaultSaveDirectory(t *testing.T) {
	t.Setenv("XDG_PICTURES_DIR", "/custom/pics")
	if DefaultSaveDirectory() != "/custom/pics/Screenshots" {
		t.Fatalf("DefaultSaveDirectory() = %q", DefaultSaveDirectory())
	}
	if ResolveSaveDirectory("") != "/custom/pics/Screenshots" {
		t.Fatalf("ResolveSaveDirectory empty = %q", ResolveSaveDirectory(""))
	}
	if ResolveSaveDirectory("/tmp/shots") != "/tmp/shots" {
		t.Fatalf("ResolveSaveDirectory abs = %q", ResolveSaveDirectory("/tmp/shots"))
	}
	if ResolveSaveDirectory("relative") != "/custom/pics/Screenshots" {
		t.Fatalf("ResolveSaveDirectory relative = %q", ResolveSaveDirectory("relative"))
	}
}

func TestSaveDirectoryFromUserDirs(t *testing.T) {
	t.Setenv("XDG_PICTURES_DIR", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	config := filepath.Join(home, ".config")
	t.Setenv("XDG_CONFIG_HOME", config)
	if err := os.MkdirAll(config, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "XDG_PICTURES_DIR=\"$HOME/My Pictures\"\n"
	if err := os.WriteFile(filepath.Join(config, "user-dirs.dirs"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "My Pictures", "Screenshots")
	if DefaultSaveDirectory() != want {
		t.Fatalf("DefaultSaveDirectory() = %q, want %q", DefaultSaveDirectory(), want)
	}
}

func TestSetSaveDirectoryRoundTrip(t *testing.T) {
	t.Setenv("XDG_PICTURES_DIR", "/pics")
	store := &Store{current: defaults(), path: filepath.Join(t.TempDir(), "settings.json")}
	if !store.SaveDirectoryIsDefault() || store.SaveDirectory() != "/pics/Screenshots" {
		t.Fatalf("default = %q defaulted=%v", store.SaveDirectory(), store.SaveDirectoryIsDefault())
	}
	if err := store.SetSaveDirectory("/tmp/lamha-shots"); err != nil {
		t.Fatal(err)
	}
	if store.SaveDirectoryIsDefault() || store.SaveDirectory() != "/tmp/lamha-shots" {
		t.Fatalf("custom = %q defaulted=%v", store.SaveDirectory(), store.SaveDirectoryIsDefault())
	}
	if err := store.SetSaveDirectory(""); err != nil {
		t.Fatal(err)
	}
	if !store.SaveDirectoryIsDefault() || store.SaveDirectory() != "/pics/Screenshots" {
		t.Fatalf("reset = %q defaulted=%v", store.SaveDirectory(), store.SaveDirectoryIsDefault())
	}
	got := settingsFromFile(settingsFile{MagnifierZoom: 4})
	if got.SaveDirectory != "" {
		t.Fatalf("missing save_directory should stay default: %q", got.SaveDirectory)
	}
}
