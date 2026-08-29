package grab

import (
	"image"
	"testing"
)

func TestMapToImageSingleMonitor(t *testing.T) {
	mon := image.Rect(1920, 0, 3840, 1080)
	screen := image.Rect(2000, 40, 2800, 800)
	got := MapToImage(screen, image.Pt(1920, 1080), mon)
	want := image.Rect(80, 40, 880, 800)
	if !got.Eq(want) {
		t.Fatalf("MapToImage() = %v, want %v", got, want)
	}
}

func TestMapToImageVirtualDesktop(t *testing.T) {
	mon := image.Rect(1920, 0, 3840, 1080)
	screen := image.Rect(2000, 40, 2800, 800)
	got := MapToImage(screen, image.Pt(3840, 1080), mon)
	if !got.Eq(screen) {
		t.Fatalf("virtual desktop MapToImage() = %v, want %v", got, screen)
	}
}

func TestFrameAtPicksSmallest(t *testing.T) {
	big := image.Rect(0, 0, 100, 100)
	small := image.Rect(10, 10, 30, 30)
	got := FrameAt([]image.Rectangle{big, small}, image.Pt(20, 20))
	if !got.Eq(small) {
		t.Fatalf("FrameAt() = %v, want small", got)
	}
	if !FrameAt([]image.Rectangle{big}, image.Pt(200, 200)).Empty() {
		t.Fatal("FrameAt outside should be empty")
	}
}

func TestAtspiStateHas(t *testing.T) {
	states := []uint32{1 << atspiStateFocused}
	if !atspiStateHas(states, atspiStateFocused) {
		t.Fatal("expected focused bit")
	}
	if atspiStateHas(states, atspiStateActive) {
		t.Fatal("did not expect active bit")
	}
	if atspiStateHas(nil, atspiStateFocused) {
		t.Fatal("empty state should not be focused")
	}
}

func TestFilterWindowsDropsTinyAndSelf(t *testing.T) {
	frames := []WindowFrame{
		{Title: "Tiny", Bounds: image.Rect(0, 0, 10, 10)},
		{Title: "Lamha capture", Class: "io.github.lamha.Lamha", Bounds: image.Rect(0, 0, 800, 600)},
		{Title: "OK", Bounds: image.Rect(0, 0, 400, 300)},
	}
	got := filterWindows(frames)
	if len(got) != 1 || got[0].Title != "OK" {
		t.Fatalf("filterWindows() = %+v", got)
	}
}
