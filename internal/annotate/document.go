// Package annotate applies ShareX-style markup to a captured image.
package annotate

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// Tool is one markup instrument.
type Tool int

const (
	ToolPen Tool = iota
	ToolBox
	ToolHighlight
	ToolBlur
	ToolStep
	ToolMagicErase
	ToolAreaErase
	// ToolSelect is used by the capture overlay and does not paint pixels.
	ToolSelect
	ToolMove
	ToolArrow
	ToolEllipse
	ToolText
)

// Color is an 8-bit sRGB color with alpha.
type Color struct {
	R, G, B, A uint8
}

// RGB builds an opaque color from a 0xRRGGBB value.
func RGB(hex uint32) Color {
	return Color{R: uint8(hex >> 16), G: uint8(hex >> 8), B: uint8(hex), A: 255}
}

func (c Color) withAlpha(alpha uint8) Color {
	c.A = alpha
	return c
}

// Point is a position in image pixels.
type Point struct {
	X, Y float64
}

// Stroke is one committed or in-progress annotation.
type Stroke struct {
	Tool   Tool
	Color  Color
	Width  float64
	Points []Point
	X1, Y1 float64
	X2, Y2 float64
	Step   int
	Text   string
}

// Document is a capture plus the annotations applied on top of it.
type Document struct {
	source  *image.NRGBA
	strokes []Stroke
	redo    []Stroke
	next    int
	raster  *image.NRGBA
	fresh   bool
}

// Open loads a screenshot from disk.
func Open(path string) (*Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open capture for annotation: %w", err)
	}
	defer file.Close()

	img, err := decodeImage(file, path)
	if err != nil {
		return nil, err
	}
	return &Document{source: toNRGBA(img), next: 1}, nil
}

func decodeImage(file *os.File, path string) (image.Image, error) {
	img, _, err := image.Decode(file)
	if err == nil {
		return img, nil
	}
	if _, seekErr := file.Seek(0, 0); seekErr != nil {
		return nil, fmt.Errorf("decode capture %s: %w", filepath.Base(path), err)
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(file)
	case ".png":
		img, err = png.Decode(file)
	}
	if err != nil {
		return nil, fmt.Errorf("decode capture %s: %w", filepath.Base(path), err)
	}
	return img, nil
}

func toNRGBA(img image.Image) *image.NRGBA {
	bounds := img.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(dst, dst.Bounds(), img, bounds.Min, draw.Src)
	return dst
}

// Size is the capture size in pixels.
func (d *Document) Size() image.Point {
	return d.source.Bounds().Size()
}

// NextStep is the number that will be placed by the next step tool click.
func (d *Document) NextStep() int {
	return d.next
}

// Add commits a finished stroke.
func (d *Document) Add(stroke Stroke) {
	if stroke.Tool == ToolPen {
		stroke.Points = SmoothPath(stroke.Points)
	}
	if stroke.Tool == ToolStep {
		if stroke.Step <= 0 {
			stroke.Step = d.next
		}
		if stroke.Step >= d.next {
			d.next = stroke.Step + 1
		}
	}
	d.strokes = append(d.strokes, stroke)
	d.redo = nil
	if d.fresh && d.raster != nil {
		apply(d.raster, stroke)
		return
	}
	d.invalidate()
}

// Undo removes the most recent stroke.
func (d *Document) Undo() bool {
	if len(d.strokes) == 0 {
		return false
	}
	removed := d.strokes[len(d.strokes)-1]
	d.strokes = d.strokes[:len(d.strokes)-1]
	d.redo = append(d.redo, removed)
	d.recomputeNext()
	d.invalidate()
	return true
}

// Redo restores the last undone stroke.
func (d *Document) Redo() bool {
	if len(d.redo) == 0 {
		return false
	}
	stroke := d.redo[len(d.redo)-1]
	d.redo = d.redo[:len(d.redo)-1]
	d.strokes = append(d.strokes, stroke)
	if stroke.Tool == ToolStep && stroke.Step >= d.next {
		d.next = stroke.Step + 1
	}
	if d.fresh && d.raster != nil {
		apply(d.raster, stroke)
		return true
	}
	d.invalidate()
	return true
}

// CanUndo reports whether a stroke can be removed.
func (d *Document) CanUndo() bool {
	return len(d.strokes) > 0
}

// CanRedo reports whether an undone stroke can be restored.
func (d *Document) CanRedo() bool {
	return len(d.redo) > 0
}

// Len is the number of committed strokes.
func (d *Document) Len() int {
	return len(d.strokes)
}

// Stroke returns a copy of the stroke at i.
func (d *Document) Stroke(i int) (Stroke, bool) {
	if i < 0 || i >= len(d.strokes) {
		return Stroke{}, false
	}
	return d.strokes[i], true
}

// Remove deletes the stroke at i.
func (d *Document) Remove(i int) bool {
	if i < 0 || i >= len(d.strokes) {
		return false
	}
	removed := d.strokes[i]
	d.strokes = append(d.strokes[:i], d.strokes[i+1:]...)
	d.redo = append(d.redo, removed)
	d.recomputeNext()
	d.invalidate()
	return true
}

// Duplicate copies the stroke at i, offset so it is easy to grab.
func (d *Document) Duplicate(i int) int {
	stroke, ok := d.Stroke(i)
	if !ok {
		return -1
	}
	stroke.Translate(24, 24)
	d.Add(stroke)
	return len(d.strokes) - 1
}

// Move translates the stroke at i.
func (d *Document) Move(i int, dx, dy float64) bool {
	if i < 0 || i >= len(d.strokes) {
		return false
	}
	d.strokes[i].Translate(dx, dy)
	d.invalidate()
	return true
}

// Replace overwrites the stroke at i without clearing redo.
func (d *Document) Replace(i int, stroke Stroke) bool {
	if i < 0 || i >= len(d.strokes) {
		return false
	}
	d.strokes[i] = stroke
	d.recomputeNext()
	d.invalidate()
	return true
}

// Hit returns the topmost stroke under p, or -1.
func (d *Document) Hit(p Point) int {
	for i := len(d.strokes) - 1; i >= 0; i-- {
		if d.strokes[i].Hit(p, 10) {
			return i
		}
	}
	return -1
}

func (d *Document) recomputeNext() {
	d.next = 1
	for _, stroke := range d.strokes {
		if stroke.Tool == ToolStep && stroke.Step >= d.next {
			d.next = stroke.Step + 1
		}
	}
}

func (d *Document) invalidate() {
	d.fresh = false
}

// Raster is the current capture with committed strokes. Do not mutate it.
func (d *Document) Raster() *image.NRGBA {
	if d.fresh && d.raster != nil {
		return d.raster
	}
	d.raster = cloneNRGBA(d.source)
	for _, stroke := range d.strokes {
		apply(d.raster, stroke)
	}
	d.fresh = true
	return d.raster
}

// Render returns a new image with every stroke applied.
func (d *Document) Render() *image.NRGBA {
	return cloneNRGBA(d.Raster())
}

// RenderExcept returns a new image with every stroke except skip applied.
func (d *Document) RenderExcept(skip int) *image.NRGBA {
	out := cloneNRGBA(d.source)
	for i, stroke := range d.strokes {
		if i == skip {
			continue
		}
		apply(out, stroke)
	}
	return out
}

func cloneNRGBA(src *image.NRGBA) *image.NRGBA {
	dst := image.NewNRGBA(src.Rect)
	copy(dst.Pix, src.Pix)
	return dst
}

// RectFromPoints builds a pixel rectangle from two image points.
func RectFromPoints(x1, y1, x2, y2 float64) image.Rectangle {
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	return image.Rect(int(math.Floor(x1)), int(math.Floor(y1)), int(math.Ceil(x2))+1, int(math.Ceil(y2))+1)
}

// Crop copies the intersection of img and region into a new image.
func Crop(img *image.NRGBA, region image.Rectangle) *image.NRGBA {
	region = region.Intersect(img.Bounds())
	if region.Empty() {
		return cloneNRGBA(img)
	}
	dst := image.NewNRGBA(image.Rect(0, 0, region.Dx(), region.Dy()))
	draw.Draw(dst, dst.Bounds(), img, region.Min, draw.Src)
	return dst
}
