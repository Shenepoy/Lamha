package launch

import (
	"errors"
	"testing"
)

func TestExecutableUsesPersistentAppImage(t *testing.T) {
	getenv := func(key string) string {
		if key == "APPIMAGE" {
			return "/opt/Lamha.AppImage"
		}
		return ""
	}
	got, err := executable(getenv, func() (string, error) {
		return "/tmp/.mount_Lamha/usr/bin/lamha", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/opt/Lamha.AppImage" {
		t.Fatalf("executable() = %q, want persistent AppImage path", got)
	}
}

func TestExecutableFallsBackToProcessPath(t *testing.T) {
	got, err := executable(func(string) string { return "" }, func() (string, error) {
		return "/usr/bin/lamha", nil
	})
	if err != nil || got != "/usr/bin/lamha" {
		t.Fatalf("executable() = %q, %v", got, err)
	}

	want := errors.New("missing executable")
	if _, err := executable(func(string) string { return "" }, func() (string, error) {
		return "", want
	}); !errors.Is(err, want) {
		t.Fatalf("executable() error = %v, want %v", err, want)
	}
}
