package grab

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestCropPNG(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 40, 20))
	path := filepath.Join(t.TempDir(), "grab.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, src); err != nil {
		t.Fatal(err)
	}
	file.Close()

	if err := CropPNG(path, image.Rect(10, 5, 30, 15)); err != nil {
		t.Fatal(err)
	}
	size, err := PNGSize(path)
	if err != nil {
		t.Fatal(err)
	}
	if size != (image.Point{X: 20, Y: 10}) {
		t.Fatalf("PNGSize() = %v, want 20x10", size)
	}
}
