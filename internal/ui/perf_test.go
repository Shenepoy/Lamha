package ui

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/lamha-app/lamha/internal/annotate"
)

func BenchmarkOverlayPreparation4K(b *testing.B) {
	path := filepath.Join(b.TempDir(), "screen.png")
	file, err := os.Create(path)
	if err != nil {
		b.Fatal(err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 3840, 2160))
	for y := 0; y < 2160; y++ {
		for x := 0; x < 3840; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8((x*13 + y*7) & 255),
				G: uint8((x*3 + y*17) & 255),
				B: uint8((x*19 + y*5) & 255),
				A: 255,
			})
		}
	}
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(file, img); err != nil {
		file.Close()
		b.Fatal(err)
	}
	if err := file.Close(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc, err := annotate.Open(path)
		if err != nil {
			b.Fatal(err)
		}
		surface := imageSurface(doc.Raster())
		if surface == nil {
			b.Fatal("imageSurface returned nil")
		}
		surface.Close()
	}
}
