package grab

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFastBackendsPreferNativeDesktop(t *testing.T) {
	t.Setenv("DESKTOP_SESSION", "")
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	if name := fastBackends()[0].name; name != "GNOME Shell" {
		t.Fatalf("GNOME first backend = %q", name)
	}
	t.Setenv("XDG_CURRENT_DESKTOP", "KDE")
	if name := fastBackends()[0].name; name != "KWin" {
		t.Fatalf("KDE first backend = %q", name)
	}
}

func TestFileNonEmpty(t *testing.T) {
	empty := filepath.Join(t.TempDir(), "empty.png")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if fileNonEmpty(empty) {
		t.Fatal("empty file should be rejected")
	}
	if fileNonEmpty(filepath.Join(t.TempDir(), "missing.png")) {
		t.Fatal("missing file should be rejected")
	}

	full := filepath.Join(t.TempDir(), "full.png")
	if err := os.WriteFile(full, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !fileNonEmpty(full) {
		t.Fatal("non-empty file should be accepted")
	}
}

func TestCopyFile(t *testing.T) {
	source := filepath.Join(t.TempDir(), "src.png")
	dest := filepath.Join(t.TempDir(), "dst.png")
	if err := os.WriteFile(source, []byte("pixels"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(source, dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "pixels" {
		t.Fatalf("copyFile() = %q", got)
	}
}
