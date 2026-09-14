package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyWithWaylandCLIForwardsMIMEAndBytes(t *testing.T) {
	directory := t.TempDir()
	argsPath := filepath.Join(directory, "args")
	dataPath := filepath.Join(directory, "data")
	commandPath := filepath.Join(directory, "wl-copy")
	command := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$CLIPBOARD_TEST_ARGS\"\nIFS= read -r line\nprintf '%s' \"$line\" > \"$CLIPBOARD_TEST_DATA\"\n"
	if err := os.WriteFile(commandPath, []byte(command), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	t.Setenv("CLIPBOARD_TEST_ARGS", argsPath)
	t.Setenv("CLIPBOARD_TEST_DATA", dataPath)

	const wantMIME = "image/png"
	wantData := []byte("png bytes")
	if err := copyWithWaylandCLI(wantData, wantMIME); err != nil {
		t.Fatalf("copyWithWaylandCLI() error = %v", err)
	}

	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.TrimSpace(string(args)), "--type\n"+wantMIME; got != want {
		t.Fatalf("wl-copy args = %q, want %q", got, want)
	}
	data, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(wantData) {
		t.Fatalf("wl-copy data = %q, want %q", data, wantData)
	}
}

func TestCopyWithWaylandCLIMissingCommand(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if err := copyWithWaylandCLI([]byte("data"), "image/png"); err == nil {
		t.Fatal("copyWithWaylandCLI() error = nil, want missing command error")
	}
}
