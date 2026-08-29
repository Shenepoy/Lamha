package ui

import (
	"image"
	"testing"

	"github.com/lamha-app/lamha/internal/grab"
)

func TestMapPickerFramesUsesMonitorOrigin(t *testing.T) {
	mon := image.Rect(1920, 0, 3840, 1080)
	frames := mapPickerFrames([]grab.WindowFrame{{
		Title:  "App",
		Bounds: image.Rect(2000, 40, 2800, 800),
	}}, mon)
	if len(frames) != 1 {
		t.Fatalf("len=%d", len(frames))
	}
	want := image.Rect(80, 40, 880, 800)
	if !frames[0].bounds.Eq(want) {
		t.Fatalf("bounds=%v want %v", frames[0].bounds, want)
	}
}

func TestPickerToScreen(t *testing.T) {
	mon := image.Rect(1920, 0, 3840, 1080)
	got := pickerToScreen(image.Rect(80, 40, 880, 800), mon)
	want := image.Rect(2000, 40, 2800, 800)
	if !got.Eq(want) {
		t.Fatalf("pickerToScreen()=%v want %v", got, want)
	}
}
