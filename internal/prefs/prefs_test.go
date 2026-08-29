package prefs

import (
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
