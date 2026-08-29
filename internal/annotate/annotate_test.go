package annotate

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestFitAndToImage(t *testing.T) {
	view := Fit(200, 100, 400, 400)
	if view.Scale != 2 {
		t.Fatalf("Fit scale = %v, want 2", view.Scale)
	}
	if view.OffsetX != 0 || view.OffsetY != 100 {
		t.Fatalf("Fit offset = (%v,%v)", view.OffsetX, view.OffsetY)
	}
	got := view.ToImage(20, 120)
	if got.X != 10 || got.Y != 10 {
		t.Fatalf("ToImage = %+v", got)
	}
}

func TestRenderBoxAndUndo(t *testing.T) {
	doc := solidDoc(24, 24, color.NRGBA{R: 10, G: 10, B: 10, A: 255})
	doc.Add(Stroke{Tool: ToolBox, Color: RGB(0xff0000), Width: 2, X1: 4, Y1: 4, X2: 12, Y2: 12})
	out := doc.Render()
	if same := out.NRGBAAt(4, 4); same.R < 200 {
		t.Fatalf("box corner = %+v, want red", same)
	}
	if !doc.Undo() {
		t.Fatal("Undo() = false")
	}
	undone := doc.Render()
	if undone.NRGBAAt(4, 4).R > 20 {
		t.Fatal("undo did not restore the original pixels")
	}
}

func TestHighlightBlends(t *testing.T) {
	doc := solidDoc(16, 16, color.NRGBA{R: 0, G: 0, B: 255, A: 255})
	doc.Add(Stroke{Tool: ToolHighlight, Color: RGB(0xffff00), Width: 4, X1: 0, Y1: 0, X2: 8, Y2: 8})
	out := doc.Render()
	pixel := out.NRGBAAt(2, 2)
	if pixel.B < 80 || pixel.G < 40 {
		t.Fatalf("highlight blend = %+v", pixel)
	}
}

func TestBlurChangesRegion(t *testing.T) {
	doc := solidDoc(20, 20, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	for x := 0; x < 20; x++ {
		doc.source.SetNRGBA(x, 10, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	}
	doc.Add(Stroke{Tool: ToolBlur, Width: 4, X1: 0, Y1: 6, X2: 19, Y2: 14})
	out := doc.Render()
	if out.NRGBAAt(10, 10).R == 255 && out.NRGBAAt(10, 8).R == 0 {
		t.Fatal("blur left the sharp white line intact")
	}
}

func TestMagicEraseRemovesSpot(t *testing.T) {
	doc := solidDoc(24, 24, color.NRGBA{R: 20, G: 80, B: 20, A: 255})
	doc.source.SetNRGBA(12, 12, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	doc.Add(Stroke{
		Tool:   ToolMagicErase,
		Width:  8,
		Points: []Point{{X: 12, Y: 12}},
	})
	out := doc.Render()
	got := out.NRGBAAt(12, 12)
	if got.R > 80 {
		t.Fatalf("magic erase left a red pixel: %+v", got)
	}
}

func TestMagicEraseRectFillsInterior(t *testing.T) {
	doc := solidDoc(40, 40, color.NRGBA{R: 0, G: 180, B: 0, A: 255})
	for y := 10; y < 30; y++ {
		for x := 10; x < 30; x++ {
			doc.source.SetNRGBA(x, y, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	doc.Add(Stroke{Tool: ToolAreaErase, X1: 10, Y1: 10, X2: 29, Y2: 29})
	out := doc.Render()
	center := out.NRGBAAt(20, 20)
	if center.R > 80 {
		t.Fatalf("area erase left the interior: %+v", center)
	}
	if center.G < 80 {
		t.Fatalf("area erase center = %+v, want surrounding green", center)
	}
}

func TestStepNumbersAdvance(t *testing.T) {
	doc := solidDoc(40, 40, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	if doc.NextStep() != 1 {
		t.Fatal("first step should be 1")
	}
	doc.Add(Stroke{Tool: ToolStep, Color: RGB(0x2563eb), Width: 8, X1: 12, Y1: 12})
	doc.Add(Stroke{Tool: ToolStep, Color: RGB(0x2563eb), Width: 8, X1: 28, Y1: 12})
	if doc.NextStep() != 3 {
		t.Fatalf("NextStep() = %d, want 3", doc.NextStep())
	}
	if !doc.Undo() || doc.NextStep() != 2 {
		t.Fatalf("after undo NextStep() = %d", doc.NextStep())
	}
}

func TestRedoRestoresStroke(t *testing.T) {
	doc := solidDoc(24, 24, color.NRGBA{R: 10, G: 10, B: 10, A: 255})
	doc.Add(Stroke{Tool: ToolBox, Color: RGB(0xff0000), Width: 2, X1: 4, Y1: 4, X2: 12, Y2: 12})
	if !doc.Undo() || !doc.CanRedo() {
		t.Fatal("expected redo after undo")
	}
	if !doc.Redo() {
		t.Fatal("Redo() = false")
	}
	if doc.Render().NRGBAAt(4, 4).R < 200 {
		t.Fatal("redo did not restore the box")
	}
}

func TestHitAndMove(t *testing.T) {
	doc := solidDoc(40, 40, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	doc.Add(Stroke{Tool: ToolStep, Color: RGB(0x2563eb), Width: 8, X1: 20, Y1: 20})
	if doc.Hit(Point{X: 20, Y: 20}) != 0 {
		t.Fatal("expected to hit the step badge")
	}
	if !doc.Move(0, 40, 0) {
		t.Fatal("Move() = false")
	}
	if doc.Hit(Point{X: 20, Y: 20}) != -1 || doc.Hit(Point{X: 60, Y: 20}) != 0 {
		t.Fatal("move did not relocate the step")
	}
}

func TestArrowAndEllipseRender(t *testing.T) {
	doc := solidDoc(48, 48, color.NRGBA{R: 8, G: 8, B: 8, A: 255})
	doc.Add(Stroke{Tool: ToolArrow, Color: RGB(0xff0000), Width: 3, X1: 6, Y1: 24, X2: 40, Y2: 24})
	doc.Add(Stroke{Tool: ToolEllipse, Color: RGB(0x00ff00), Width: 3, X1: 8, Y1: 8, X2: 28, Y2: 22})
	out := doc.Render()
	if out.NRGBAAt(20, 24).R < 100 {
		t.Fatal("arrow shaft was not drawn")
	}
	if out.NRGBAAt(8, 15).G < 80 && out.NRGBAAt(18, 8).G < 80 {
		t.Fatal("ellipse outline was not drawn")
	}
}

func TestTextAndDuplicate(t *testing.T) {
	doc := solidDoc(80, 40, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	doc.Add(Stroke{Tool: ToolText, Color: RGB(0xffffff), Width: 8, X1: 4, Y1: 6, Text: "Hi"})
	if doc.Hit(Point{X: 8, Y: 10}) != 0 {
		t.Fatal("expected to hit the text")
	}
	if doc.Duplicate(0) != 1 {
		t.Fatal("Duplicate() did not append")
	}
	if doc.Hit(Point{X: 32, Y: 34}) != 1 {
		t.Fatal("duplicated text was not offset")
	}
}

func TestArabicTextRenders(t *testing.T) {
	doc := solidDoc(160, 56, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	doc.Add(Stroke{Tool: ToolText, Color: RGB(0xffffff), Width: 8, X1: 10, Y1: 8, Text: "مرحبا"})
	out := doc.Render()
	painted := false
	for y := 0; y < 56 && !painted; y++ {
		for x := 0; x < 160; x++ {
			if out.NRGBAAt(x, y).R > 40 {
				painted = true
				break
			}
		}
	}
	if !painted {
		t.Fatal("Arabic text was not painted")
	}
	if doc.Hit(Point{X: 18, Y: 18}) != 0 {
		t.Fatal("expected to hit the Arabic text")
	}
}

func TestRenderExceptSkipsStroke(t *testing.T) {
	doc := solidDoc(20, 20, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	doc.Add(Stroke{Tool: ToolBox, Color: RGB(0xff0000), Width: 2, X1: 2, Y1: 2, X2: 8, Y2: 8})
	doc.Add(Stroke{Tool: ToolBox, Color: RGB(0x00ff00), Width: 2, X1: 10, Y1: 10, X2: 16, Y2: 16})
	full := doc.Render()
	withoutFirst := doc.RenderExcept(0)
	if full.NRGBAAt(3, 3).R < 100 {
		t.Fatal("full render missing first box")
	}
	if withoutFirst.NRGBAAt(3, 3).R > 40 {
		t.Fatal("RenderExcept(0) still drew the first box")
	}
	if withoutFirst.NRGBAAt(11, 11).G < 100 {
		t.Fatal("RenderExcept(0) dropped the second box")
	}
}

func TestCrop(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	src.SetNRGBA(7, 3, color.NRGBA{R: 12, G: 13, B: 14, A: 255})
	got := Crop(src, image.Rect(5, 2, 9, 6))
	if got.Bounds().Dx() != 4 || got.Bounds().Dy() != 4 {
		t.Fatalf("Crop size = %v", got.Bounds())
	}
	if got.NRGBAAt(2, 1).R != 12 {
		t.Fatalf("Crop pixel = %+v", got.NRGBAAt(2, 1))
	}
}

func TestOpenRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shot.png")
	src := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	src.SetNRGBA(3, 3, color.NRGBA{R: 9, G: 8, B: 7, A: 255})
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, src); err != nil {
		t.Fatal(err)
	}
	file.Close()

	doc, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Size() != (image.Point{X: 8, Y: 8}) {
		t.Fatalf("Size() = %v", doc.Size())
	}
	if doc.Render().NRGBAAt(3, 3).R != 9 {
		t.Fatal("opened capture lost pixel data")
	}
}

func solidDoc(w, h int, fill color.NRGBA) *Document {
	src := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			src.SetNRGBA(x, y, fill)
		}
	}
	return &Document{source: src, next: 1}
}
