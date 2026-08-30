package annotate

import (
	"image"
	"image/color"
	"math"
)

func apply(dst *image.NRGBA, stroke Stroke) {
	width := math.Max(stroke.Width, 1)
	switch stroke.Tool {
	case ToolPen:
		stampPath(dst, stroke.Points, width/2, stroke.Color.nrgbaColor())
	case ToolBox:
		strokeRect(dst, rectFrom(stroke), int(math.Round(width)), stroke.Color.nrgbaColor())
	case ToolHighlight:
		fillRectBlend(dst, rectFrom(stroke), stroke.Color.withAlpha(96).nrgbaColor())
	case ToolBlur:
		boxBlur(dst, rectFrom(stroke), int(math.Max(2, math.Round(width))))
	case ToolStep:
		x, y := stroke.X1, stroke.Y1
		if len(stroke.Points) > 0 {
			x, y = stroke.Points[0].X, stroke.Points[0].Y
		}
		drawStep(dst, x, y, stroke.Step, stroke.Color.nrgbaColor(), StepDiameter(width))
	case ToolMagicErase:
		magicErase(dst, stroke.Points, width)
	case ToolAreaErase:
		magicEraseRect(dst, rectFrom(stroke))
	case ToolArrow:
		strokeArrow(dst, stroke.X1, stroke.Y1, stroke.X2, stroke.Y2, width, stroke.Color.nrgbaColor())
	case ToolEllipse:
		strokeEllipse(dst, stroke.X1, stroke.Y1, stroke.X2, stroke.Y2, width, stroke.Color.nrgbaColor())
	case ToolText:
		if stroke.Text != "" {
			drawLabel(dst, stroke.X1, stroke.Y1, stroke.Text, stroke.Color.nrgbaColor(), width)
		}
	case ToolSelect, ToolMove:
		// Overlay-only tools.
	}
}

func (c Color) nrgbaColor() color.NRGBA {
	return color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

func rectFrom(stroke Stroke) image.Rectangle {
	x1, x2 := stroke.X1, stroke.X2
	y1, y2 := stroke.Y1, stroke.Y2
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	return image.Rect(int(math.Floor(x1)), int(math.Floor(y1)), int(math.Ceil(x2))+1, int(math.Ceil(y2))+1)
}

func stampPath(dst *image.NRGBA, points []Point, radius float64, c color.NRGBA) {
	if radius < 0.5 {
		radius = 0.5
	}
	var prev Point
	for i, point := range points {
		if i == 0 {
			stampDisc(dst, point.X, point.Y, radius, c)
			prev = point
			continue
		}
		dx := point.X - prev.X
		dy := point.Y - prev.Y
		dist := math.Hypot(dx, dy)
		steps := int(math.Ceil(dist / math.Max(radius*0.45, 1)))
		if steps < 1 {
			steps = 1
		}
		for s := 1; s <= steps; s++ {
			t := float64(s) / float64(steps)
			stampDisc(dst, prev.X+dx*t, prev.Y+dy*t, radius, c)
		}
		prev = point
	}
}

func stampDisc(dst *image.NRGBA, cx, cy, radius float64, c color.NRGBA) {
	minX := int(math.Floor(cx - radius - 1))
	maxX := int(math.Ceil(cx + radius + 1))
	minY := int(math.Floor(cy - radius - 1))
	maxY := int(math.Ceil(cy + radius + 1))
	feather := math.Max(1.1, radius*0.18)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if !inBounds(dst, x, y) {
				continue
			}
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			d := math.Hypot(dx, dy)
			if d > radius+feather {
				continue
			}
			coverage := 1.0
			if d > radius-feather {
				t := (d - (radius - feather)) / (2 * feather)
				coverage = 1 - t
				if coverage < 0 {
					continue
				}
			}
			ink := c
			ink.A = uint8(float64(c.A) * coverage)
			blendAt(dst, x, y, ink)
		}
	}
}

func strokeRect(dst *image.NRGBA, r image.Rectangle, width int, c color.NRGBA) {
	r = r.Intersect(dst.Bounds())
	if r.Empty() || width < 1 {
		return
	}
	fillRect(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, min(r.Min.Y+width, r.Max.Y)), c)
	fillRect(dst, image.Rect(r.Min.X, max(r.Max.Y-width, r.Min.Y), r.Max.X, r.Max.Y), c)
	fillRect(dst, image.Rect(r.Min.X, r.Min.Y, min(r.Min.X+width, r.Max.X), r.Max.Y), c)
	fillRect(dst, image.Rect(max(r.Max.X-width, r.Min.X), r.Min.Y, r.Max.X, r.Max.Y), c)
}

func fillRect(dst *image.NRGBA, r image.Rectangle, c color.NRGBA) {
	r = r.Intersect(dst.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			blendAt(dst, x, y, c)
		}
	}
}

func fillRectBlend(dst *image.NRGBA, r image.Rectangle, c color.NRGBA) {
	fillRect(dst, r.Intersect(dst.Bounds()), c)
}

func inBounds(img *image.NRGBA, x, y int) bool {
	return image.Pt(x, y).In(img.Rect)
}

func blendAt(dst *image.NRGBA, x, y int, src color.NRGBA) {
	if src.A == 255 {
		dst.SetNRGBA(x, y, src)
		return
	}
	if src.A == 0 {
		return
	}
	dstC := dst.NRGBAAt(x, y)
	a := uint32(src.A)
	ia := 255 - a
	dst.SetNRGBA(x, y, color.NRGBA{
		R: uint8((uint32(src.R)*a + uint32(dstC.R)*ia) / 255),
		G: uint8((uint32(src.G)*a + uint32(dstC.G)*ia) / 255),
		B: uint8((uint32(src.B)*a + uint32(dstC.B)*ia) / 255),
		A: 255,
	})
}
