package capture

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func BenchmarkSaveImage4K(b *testing.B) {
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
	store, err := NewStore(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.SaveImage(img); err != nil {
			b.Fatal(err)
		}
	}
}

func TestSaveURI(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "portal image.png")
	contents := []byte("not a real image, but a valid copy test")
	if err := os.WriteFile(source, contents, 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := NewStore(filepath.Join(directory, "captures"))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := store.SaveURI(fileURI(source))
	if err != nil {
		t.Fatalf("SaveURI() error = %v", err)
	}
	if filepath.Dir(saved.Path) != store.Directory() {
		t.Fatalf("SaveURI() path directory = %q, want %q", filepath.Dir(saved.Path), store.Directory())
	}
	got, err := os.ReadFile(saved.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(contents) {
		t.Fatalf("SaveURI() contents = %q, want %q", got, contents)
	}
}

func TestSaveImage(t *testing.T) {
	directory := t.TempDir()
	store, err := NewStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	saved, err := store.SaveImage(img)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(saved.Path) != store.Directory() {
		t.Fatalf("SaveImage() directory = %q", filepath.Dir(saved.Path))
	}
	if saved.Width != 3 || saved.Height != 3 || saved.Bytes <= 0 {
		t.Fatalf("SaveImage() metadata = %dx%d, %d bytes", saved.Width, saved.Height, saved.Bytes)
	}
	file, err := os.Open(saved.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := png.Decode(file); err != nil {
		t.Fatalf("SaveImage() did not write a PNG: %v", err)
	}
}

func TestWritePNG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "annotated.png")
	src := []byte("placeholder")
	if err := os.WriteFile(path, src, 0o644); err != nil {
		t.Fatal(err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	if err := WritePNG(path, img); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	decoded, err := png.Decode(file)
	if err != nil {
		t.Fatalf("WritePNG() did not write a PNG: %v", err)
	}
	if decoded.Bounds().Dx() != 2 || decoded.Bounds().Dy() != 2 {
		t.Fatalf("WritePNG() size = %v", decoded.Bounds())
	}
}

func TestListOrdersNewestFirst(t *testing.T) {
	directory := t.TempDir()
	store, err := NewStore(directory)
	if err != nil {
		t.Fatal(err)
	}

	older := filepath.Join(directory, "older.png")
	newer := filepath.Join(directory, "newer.png")
	skipped := filepath.Join(directory, "notes.txt")
	if err := os.WriteFile(older, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newer, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skipped, []byte("no"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(older, time.Unix(100, 0), time.Unix(100, 0)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newer, time.Unix(200, 0), time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}

	got, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("List() len = %d, want 2", len(got))
	}
	if filepath.Base(got[0].Path) != "newer.png" || filepath.Base(got[1].Path) != "older.png" {
		t.Fatalf("List() order = %q, %q", got[0].Path, got[1].Path)
	}
}

func TestPathFromFileURI(t *testing.T) {
	path, err := pathFromFileURI("file:///tmp/a%20capture.png")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/tmp/a capture.png" {
		t.Fatalf("pathFromFileURI() = %q", path)
	}

	if _, err := pathFromFileURI("https://example.test/capture.png"); err == nil {
		t.Fatal("pathFromFileURI() accepted a non-file URI")
	}
}
