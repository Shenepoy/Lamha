package version

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var calver = regexp.MustCompile(`^[0-9]{2}\.[0-9]{2}\.[0-9]+$`)

func TestVERSIONFileIsCalVer(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	name := strings.TrimSpace(string(raw))
	if !calver.MatchString(name) {
		t.Fatalf("VERSION = %q, want YY.0M.MICRO", name)
	}
}

func TestSourceBuildVersionMatchesVERSION(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(raw))
	if Version != want {
		t.Fatalf("source-build Version = %q, VERSION = %q", Version, want)
	}
}
