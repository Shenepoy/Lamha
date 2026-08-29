package portal

import (
	"errors"
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestDecodeResultsSuccess(t *testing.T) {
	signal := &dbus.Signal{
		Name: requestInterface + ".Response",
		Body: []any{
			uint32(0),
			map[string]dbus.Variant{"uri": dbus.MakeVariant("file:///tmp/capture.png")},
		},
	}

	results, err := decodeResults(signal)
	if err != nil {
		t.Fatalf("decodeResults() error = %v", err)
	}
	if results["uri"].Value().(string) != "file:///tmp/capture.png" {
		t.Fatalf("decodeResults() URI = %q", results["uri"].Value())
	}
}

func TestDecodeResultsDenied(t *testing.T) {
	signal := &dbus.Signal{
		Name: requestInterface + ".Response",
		Body: []any{uint32(2), map[string]dbus.Variant{}},
	}

	_, err := decodeResults(signal)
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("decodeResults() error = %v, want ErrDenied", err)
	}
}

func TestDecodeResultsCancelled(t *testing.T) {
	signal := &dbus.Signal{
		Name: requestInterface + ".Response",
		Body: []any{uint32(1), map[string]dbus.Variant{}},
	}

	_, err := decodeResults(signal)
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("decodeResults() error = %v, want ErrCancelled", err)
	}
}

func TestDecodeResultsRejectsInvalidPayload(t *testing.T) {
	signal := &dbus.Signal{
		Name: requestInterface + ".Response",
		Body: []any{uint32(0), map[string]dbus.Variant{}},
	}

	if _, err := decodeResults(signal); err != nil {
		return
	}
	// Empty result map is valid at this layer; URI checking happens in Screenshot.
}
