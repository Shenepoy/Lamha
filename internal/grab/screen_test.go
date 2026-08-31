package grab

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLiveFastCaptureTiming(t *testing.T) {
	if os.Getenv("LAMHA_LIVE_CAPTURE") == "" {
		t.Skip("set LAMHA_LIVE_CAPTURE=1 to time the active desktop backend")
	}
	started := time.Now()
	path, err := Fast(context.Background())
	elapsed := time.Since(started)
	if path != "" {
		defer os.Remove(path)
	}
	if elapsed >= 500*time.Millisecond {
		t.Fatalf("Fast() stalled for %s before portal fallback: %v", elapsed, err)
	}
	t.Logf("Fast() completed in %s (result: %v)", elapsed, err)
}

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

func TestFastBackendsSkipGNOMEScreenshotOnWayland(t *testing.T) {
	t.Setenv("DESKTOP_SESSION", "gnome")
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	for _, candidate := range fastBackends() {
		if candidate.name == "gnome-screenshot" {
			t.Fatal("gnome-screenshot should not delay portal fallback on Wayland")
		}
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
