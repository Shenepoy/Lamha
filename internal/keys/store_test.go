package keys

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsCoverCatalog(t *testing.T) {
	got := defaults()
	for _, item := range Catalog() {
		if got[item.ID] != item.Default {
			t.Fatalf("defaults[%s] = %q, want %q", item.ID, got[item.ID], item.Default)
		}
	}
}

func TestSetRejectsConflicts(t *testing.T) {
	store := &Store{values: defaults(), path: filepath.Join(t.TempDir(), "keybinds.json")}
	if err := store.Set(ToolPen, "S"); err == nil {
		t.Fatal("Set() conflict error = nil")
	}
	if store.Accel(ToolPen) != "P" {
		t.Fatalf("Accel(ToolPen) = %q after rejected set", store.Accel(ToolPen))
	}
}

func TestSetAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	store := &Store{values: defaults(), path: filepath.Join(dir, "lamha", "keybinds.json")}
	if err := store.Set(CaptureArea, "<Control><Alt>X"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	loaded := Load()
	if loaded.Accel(CaptureArea) != "<Control><Alt>X" {
		t.Fatalf("Load() capture-area = %q", loaded.Accel(CaptureArea))
	}
	if loaded.Accel(ToolPen) != "P" {
		t.Fatalf("Load() tool-pen = %q", loaded.Accel(ToolPen))
	}
}

func TestResetAll(t *testing.T) {
	store := &Store{values: defaults(), path: filepath.Join(t.TempDir(), "keybinds.json")}
	if err := store.Set(ToolPen, "D"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := store.ResetAll(); err != nil {
		t.Fatalf("ResetAll() error = %v", err)
	}
	if store.Accel(ToolPen) != "P" {
		t.Fatalf("Accel(ToolPen) = %q after reset", store.Accel(ToolPen))
	}
}

func TestEmptyAccelUnbinds(t *testing.T) {
	store := &Store{values: defaults(), path: filepath.Join(t.TempDir(), "keybinds.json")}
	if err := store.Set(CaptureScreen, ""); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if store.Accel(CaptureScreen) != "" {
		t.Fatalf("Accel(CaptureScreen) = %q, want empty", store.Accel(CaptureScreen))
	}
	_ = os.Remove(store.path)
}
