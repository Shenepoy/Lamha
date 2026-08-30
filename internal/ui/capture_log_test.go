package ui

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestFindCenteredBrightSquare(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 400, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 400; x++ {
			img.Set(x, y, color.RGBA{R: 20, G: 20, B: 22, A: 255})
		}
	}
	if _, found := findCenteredBrightSquare(img, 32, 180); found {
		t.Fatal("dark frame should not report a white square")
	}

	for y := 110; y < 190; y++ {
		for x := 160; x < 240; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	got, found := findCenteredBrightSquare(img, 32, 180)
	if !found {
		t.Fatal("expected the centered white square")
	}
	if got.Dx() < 64 || got.Dy() < 64 {
		t.Fatalf("square too small: %v", got)
	}
}

func TestFlashFramesFromEnv(t *testing.T) {
	dir := os.Getenv("LAMHA_FLASH_DIR")
	if dir == "" {
		t.Skip("set LAMHA_FLASH_DIR to inspect grim frames")
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.png"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatalf("no png frames in %s", dir)
	}
	found := false
	for _, path := range matches {
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if r, ok := findCenteredBrightSquare(img, 48, 220); ok {
			t.Logf("%s white square %v", filepath.Base(path), r)
			found = true
		}
	}
	if found {
		t.Fatal("centered white square present in capture frames")
	}
}
