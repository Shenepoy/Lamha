package annotate

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
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

func TestAdjustStepRenumberAndDeleteZero(t *testing.T) {
	doc := addSteps(solidDoc(160, 40, color.NRGBA{R: 255, G: 255, B: 255, A: 255}), 6)
	if doc.AdjustStep(2, -1, true, true) < 0 {
		t.Fatal("minus on 3 should keep that step")
	}
	if got := stepNumbers(doc); got != "1,2,3,4,5" {
		t.Fatalf("reduce 3 to 2 should drop every number and delete 1: %s", got)
	}

	doc = addSteps(solidDoc(80, 40, color.NRGBA{R: 255, G: 255, B: 255, A: 255}), 3)
	if doc.AdjustStep(1, 1, true, true) != 1 {
		t.Fatal("plus should keep the step")
	}
	if got := stepNumbers(doc); got != "2,3,4" {
		t.Fatalf("renumber plus = %s", got)
	}
	if doc.AdjustStep(1, -1, true, true) != 1 {
		t.Fatal("minus should keep that step")
	}
	if got := stepNumbers(doc); got != "1,2,3" {
		t.Fatalf("renumber minus = %s", got)
	}
	if doc.AdjustStep(0, -1, true, true) != -1 {
		t.Fatal("minus to 0 should delete")
	}
	if got := stepNumbers(doc); got != "1,2" {
		t.Fatalf("delete zero + renumber = %s", got)
	}
	if doc.NextStep() != 3 {
		t.Fatalf("NextStep() = %d after delete", doc.NextStep())
	}

	doc = solidDoc(80, 40, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	doc.Add(Stroke{Tool: ToolStep, Color: RGB(0x2563eb), Width: 8, X1: 10, Y1: 10})
	doc.Add(Stroke{Tool: ToolStep, Color: RGB(0x2563eb), Width: 8, X1: 30, Y1: 10})
	if doc.AdjustStep(0, 1, false, false) != 0 {
		t.Fatal("isolated plus")
	}
	if got := stepNumbers(doc); got != "2,2" {
		t.Fatalf("no renumber = %s", got)
	}
	if doc.AdjustStep(0, -2, false, false) != 0 {
		t.Fatal("isolated minus to zero")
	}
	stroke, ok := doc.Stroke(0)
	if !ok || stroke.Step != 0 {
		t.Fatalf("kept zero step: %+v", stroke)
	}
}

func addSteps(doc *Document, n int) *Document {
	for i := 0; i < n; i++ {
		doc.Add(Stroke{Tool: ToolStep, Color: RGB(0xe11d48), Width: 8, X1: float64(12 + i*20), Y1: 16})
	}
	return doc
}

func stepNumbers(doc *Document) string {
	parts := make([]string, 0, doc.Len())
	for i := 0; i < doc.Len(); i++ {
		stroke, ok := doc.Stroke(i)
		if !ok || stroke.Tool != ToolStep {
			continue
		}
		parts = append(parts, fmt.Sprintf("%d", stroke.Step))
	}
	return strings.Join(parts, ",")
}

func TestStepBadgeIsSolid(t *testing.T) {
	doc := solidDoc(80, 80, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	doc.Add(Stroke{Tool: ToolStep, Color: RGB(0xe11d48), Width: 8, X1: 40, Y1: 40})
	out := doc.Render()
	rim := out.NRGBAAt(58, 40)
	if rim.R < 100 || rim.G > 90 {
		t.Fatalf("step rim = %+v, want a solid red badge", rim)
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

func TestRasterWithoutStrokesReusesSourceUntilFirstMark(t *testing.T) {
	doc := solidDoc(20, 20, color.NRGBA{R: 4, G: 5, B: 6, A: 255})
	if got := doc.Raster(); got != doc.source {
		t.Fatal("Raster copied an unchanged screenshot")
	}
	doc.Add(Stroke{Tool: ToolBox, Color: RGB(0xff0000), Width: 2, X1: 2, Y1: 2, X2: 8, Y2: 8})
	if got := doc.Raster(); got == doc.source {
		t.Fatal("Raster mutated the source after adding a mark")
	}
	if got := doc.source.NRGBAAt(2, 2); got.R != 4 {
		t.Fatalf("source pixel was changed: %+v", got)
	}
}

func TestRenderRegionWithoutStrokesAvoidsFullFrame(t *testing.T) {
	doc := solidDoc(100, 80, color.NRGBA{R: 4, G: 5, B: 6, A: 255})
	doc.source.SetNRGBA(42, 31, color.NRGBA{R: 90, G: 80, B: 70, A: 255})
	got := doc.RenderRegion(image.Rect(40, 30, 50, 36))
	if got.Bounds() != image.Rect(0, 0, 10, 6) {
		t.Fatalf("RenderRegion bounds = %v", got.Bounds())
	}
	if pixel := got.NRGBAAt(2, 1); pixel.R != 90 || pixel.G != 80 || pixel.B != 70 {
		t.Fatalf("RenderRegion pixel = %+v", pixel)
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
