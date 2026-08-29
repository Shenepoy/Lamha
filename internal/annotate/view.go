package annotate

import "math"

// View maps a fitted image into a widget.
type View struct {
	Scale   float64
	OffsetX float64
	OffsetY float64
	ImageW  int
	ImageH  int
}

// Fit letterboxes the image inside the drawing area.
func Fit(imageW, imageH, areaW, areaH int) View {
	if imageW < 1 || imageH < 1 || areaW < 1 || areaH < 1 {
		return View{Scale: 1, ImageW: imageW, ImageH: imageH}
	}
	scale := math.Min(float64(areaW)/float64(imageW), float64(areaH)/float64(imageH))
	drawW := float64(imageW) * scale
	drawH := float64(imageH) * scale
	return View{
		Scale:   scale,
		OffsetX: (float64(areaW) - drawW) / 2,
		OffsetY: (float64(areaH) - drawH) / 2,
		ImageW:  imageW,
		ImageH:  imageH,
	}
}

// ToWidget converts image pixels into widget coordinates.
func (v View) ToWidget(x, y float64) (wx, wy float64) {
	return x*v.Scale + v.OffsetX, y*v.Scale + v.OffsetY
}

// ToImage converts widget coordinates into image pixels and clamps them.
func (v View) ToImage(x, y float64) Point {
	if v.Scale == 0 {
		return Point{}
	}
	ix := (x - v.OffsetX) / v.Scale
	iy := (y - v.OffsetY) / v.Scale
	return Point{
		X: clamp(ix, 0, float64(max(v.ImageW-1, 0))),
		Y: clamp(iy, 0, float64(max(v.ImageH-1, 0))),
	}
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
