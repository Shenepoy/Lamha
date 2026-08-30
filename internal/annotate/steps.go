package annotate

import (
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotk4/pkg/pangocairo"
)

func drawStep(dst *image.NRGBA, cx, cy float64, number int, c color.NRGBA, diameter float64) {
	if number < 0 {
		number = 0
	}
	if diameter < 16 {
		diameter = 16
	}
	label := strconv.Itoa(number)
	if drawCairoStep(dst, cx, cy, label, c, diameter, true) {
		return
	}
	drawCairoStep(dst, cx, cy, label, c, diameter, false)
}

func drawCairoStep(dst *image.NRGBA, cx, cy float64, label string, c color.NRGBA, diameter float64, usePango bool) bool {
	pad := 2
	size := int(math.Round(diameter)) + pad*2
	if size%2 == 0 {
		size++
	}
	surf := cairo.CreateImageSurface(cairo.FormatARGB32, size, size)
	if surf == nil {
		return false
	}
	cr := cairo.Create(surf)
	if cr == nil {
		return false
	}
	cr.SetAntialias(cairo.AntialiasGray)

	mid := float64(size) / 2
	radius := math.Floor(diameter/2) - 0.5
	if radius < 4 {
		radius = 4
	}
	cr.SetSourceRGBA(float64(c.R)/255, float64(c.G)/255, float64(c.B)/255, float64(c.A)/255)
	cr.Arc(mid, mid, radius, 0, 2*math.Pi)
	cr.Fill()

	ink := numberInk(c)
	fontSize := stepFontSize(diameter, label)
	cr.SetSourceRGBA(float64(ink.R)/255, float64(ink.G)/255, float64(ink.B)/255, 1)
	if usePango {
		if !paintStepPango(cr, mid, label, fontSize, radius) {
			return false
		}
	} else {
		paintStepToyFont(cr, mid, label, fontSize)
	}

	ox := int(math.Round(cx)) - size/2
	oy := int(math.Round(cy)) - size/2
	blitCairoOver(dst, ox, oy, surf)
	return true
}

func paintStepPango(cr *cairo.Context, mid float64, label string, fontSize, radius float64) bool {
	layout := newStepLayout(cr, label, fontSize)
	if layout == nil {
		return false
	}
	ix, iy, iw, ih, ok := copyLayoutInk(layout)
	if !ok {
		return false
	}
	max := radius * 1.35
	if float64(iw) > max || float64(ih) > max {
		scale := max / math.Max(float64(iw), float64(ih))
		layout = newStepLayout(cr, label, fontSize*scale)
		if layout == nil {
			return false
		}
		ix, iy, iw, ih, ok = copyLayoutInk(layout)
		if !ok {
			return false
		}
	}
	cr.MoveTo(mid-float64(ix)-float64(iw)/2+stepOpticalShift(label, float64(iw)), mid-float64(iy)-float64(ih)/2)
	pangocairo.ShowLayout(cr, layout)
	return true
}

func stepOpticalShift(label string, inkW float64) float64 {
	if label == "1" {
		return -inkW * 0.18
	}
	return 0
}

func copyLayoutInk(layout *pango.Layout) (x, y, w, h int, ok bool) {
	ink, _ := layout.PixelExtents()
	if ink == nil {
		return 0, 0, 0, 0, false
	}
	x, y, w, h = ink.X(), ink.Y(), ink.Width(), ink.Height()
	return x, y, w, h, w > 0 && h > 0
}

func paintStepToyFont(cr *cairo.Context, mid float64, label string, fontSize float64) {
	cr.SelectFontFace("sans-serif", cairo.FontSlantNormal, cairo.FontWeightBold)
	cr.SetFontSize(fontSize)
	ext := cr.TextExtents(label)
	cr.MoveTo(mid-ext.Width/2-ext.XBearing+stepOpticalShift(label, ext.Width), mid-ext.Height/2-ext.YBearing)
	cr.ShowText(label)
}

func newStepLayout(cr *cairo.Context, text string, size float64) *pango.Layout {
	if cr == nil {
		return nil
	}
	layout := pangocairo.CreateLayout(cr)
	if layout == nil {
		return nil
	}
	desc := pango.FontDescriptionFromString("Noto Sans, DejaVu Sans, sans-serif")
	desc.SetAbsoluteSize(math.Max(12, size) * pango.SCALE)
	desc.SetWeight(pango.WeightBold)
	layout.SetFontDescription(desc)
	layout.SetText(text)
	return layout
}

func stepFontSize(diameter float64, label string) float64 {
	if len(label) > 1 {
		return diameter * 0.44
	}
	return diameter * 0.58
}

func numberInk(c color.NRGBA) color.NRGBA {
	lum := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
	if lum > 176 {
		return color.NRGBA{R: 28, G: 28, B: 30, A: 255}
	}
	return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
}
