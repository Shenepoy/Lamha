package annotate

import (
	"image"
	"image/color"
	"math"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotk4/pkg/pangocairo"
)

const textFontFamily = "Noto Sans Arabic, Noto Naskh Arabic, Noto Sans, DejaVu Sans, sans-serif"

func newTextLayout(cr *cairo.Context, text string, size float64) *pango.Layout {
	layout := pangocairo.CreateLayout(cr)
	layout.SetAutoDir(true)
	desc := pango.FontDescriptionFromString(textFontFamily)
	desc.SetAbsoluteSize(math.Max(10, size) * pango.SCALE)
	desc.SetWeight(pango.WeightBold)
	layout.SetFontDescription(desc)
	layout.SetText(text)
	return layout
}

func measurePangoText(text string, width float64) (w, h int, ok bool) {
	if text == "" {
		text = " "
	}
	surf := cairo.CreateImageSurface(cairo.FormatARGB32, 1, 1)
	if surf == nil {
		return 0, 0, false
	}
	cr := cairo.Create(surf)
	if cr == nil {
		return 0, 0, false
	}
	pw, ph := newTextLayout(cr, text, TextSize(width)).PixelSize()
	if pw < 1 || ph < 1 {
		return 0, 0, false
	}
	return pw, ph, true
}

func pangoTextBounds(x, y float64, text string, width float64) (image.Rectangle, bool) {
	w, h, ok := measurePangoText(text, width)
	if !ok {
		return image.Rectangle{}, false
	}
	pad := math.Max(3, TextSize(width)*0.12)
	return RectFromPoints(x-pad, y-pad, x+float64(w)+pad, y+float64(h)+pad), true
}

// PaintText draws shaped Unicode, including Arabic, into a cairo context.
func PaintText(cr *cairo.Context, x, y float64, text string, r, g, b float64, width float64) {
	if text == "" || cr == nil {
		return
	}
	cr.Save()
	cr.SetSourceRGB(r, g, b)
	cr.MoveTo(x, y)
	pangocairo.ShowLayout(cr, newTextLayout(cr, text, TextSize(width)))
	cr.Restore()
}

func drawPangoLabel(dst *image.NRGBA, x, y float64, text string, ink color.NRGBA, width float64) bool {
	if text == "" || dst == nil {
		return false
	}
	w, h, ok := measurePangoText(text, width)
	if !ok {
		return false
	}
	pad := 4
	surf := cairo.CreateImageSurface(cairo.FormatARGB32, w+pad*2, h+pad*2)
	if surf == nil {
		return false
	}
	cr := cairo.Create(surf)
	if cr == nil {
		return false
	}
	cr.SetSourceRGBA(float64(ink.R)/255, float64(ink.G)/255, float64(ink.B)/255, float64(ink.A)/255)
	cr.MoveTo(float64(pad), float64(pad))
	pangocairo.ShowLayout(cr, newTextLayout(cr, text, TextSize(width)))
	blitCairoOver(dst, int(math.Round(x))-pad, int(math.Round(y))-pad, surf)
	return true
}

func blitCairoOver(dst *image.NRGBA, ox, oy int, surf *cairo.Surface) {
	if dst == nil || surf == nil {
		return
	}
	surf.Flush()
	data := surf.Data()
	stride := surf.Stride()
	w, h := surf.Width(), surf.Height()
	bounds := dst.Bounds()
	for row := 0; row < h; row++ {
		dy := oy + row
		if dy < bounds.Min.Y || dy >= bounds.Max.Y {
			continue
		}
		off := row * stride
		if off < 0 || off >= len(data) {
			continue
		}
		line := data[off:]
		for col := 0; col < w; col++ {
			dx := ox + col
			if dx < bounds.Min.X || dx >= bounds.Max.X {
				continue
			}
			i := col * 4
			if i+3 >= len(line) {
				break
			}
			b, g, r, a := line[i], line[i+1], line[i+2], line[i+3]
			if a == 0 {
				continue
			}
			if a < 255 {
				ia := int(a)
				r = uint8((int(r)*255 + ia/2) / ia)
				g = uint8((int(g)*255 + ia/2) / ia)
				b = uint8((int(b)*255 + ia/2) / ia)
			}
			dst.SetNRGBA(dx, dy, overNRGBA(dst.NRGBAAt(dx, dy), color.NRGBA{R: r, G: g, B: b, A: a}))
		}
	}
}

func overNRGBA(dst, src color.NRGBA) color.NRGBA {
	if src.A == 255 || dst.A == 0 {
		return src
	}
	if src.A == 0 {
		return dst
	}
	sa := uint32(src.A)
	inv := 255 - sa
	return color.NRGBA{
		R: uint8((uint32(src.R)*sa + uint32(dst.R)*inv) / 255),
		G: uint8((uint32(src.G)*sa + uint32(dst.G)*inv) / 255),
		B: uint8((uint32(src.B)*sa + uint32(dst.B)*inv) / 255),
		A: uint8(sa + uint32(dst.A)*inv/255),
	}
}
