package annotate

import (
	"image"
	"math"
)

// Translate moves every coordinate in the stroke.
func (s *Stroke) Translate(dx, dy float64) {
	s.X1 += dx
	s.Y1 += dy
	s.X2 += dx
	s.Y2 += dy
	for i := range s.Points {
		s.Points[i].X += dx
		s.Points[i].Y += dy
	}
}

// Bounds is the pixel rectangle that contains the stroke.
func (s Stroke) Bounds() image.Rectangle {
	switch s.Tool {
	case ToolPen, ToolMagicErase:
		if len(s.Points) == 0 {
			return image.Rectangle{}
		}
		minX, minY := s.Points[0].X, s.Points[0].Y
		maxX, maxY := minX, minY
		for _, p := range s.Points[1:] {
			minX = math.Min(minX, p.X)
			minY = math.Min(minY, p.Y)
			maxX = math.Max(maxX, p.X)
			maxY = math.Max(maxY, p.Y)
		}
		pad := math.Max(s.Width/2, 4)
		return RectFromPoints(minX-pad, minY-pad, maxX+pad, maxY+pad)
	case ToolStep:
		x, y := s.X1, s.Y1
		if len(s.Points) > 0 {
			x, y = s.Points[0].X, s.Points[0].Y
		}
		r := stepDiameter(s.Width) / 2
		return RectFromPoints(x-r, y-r, x+r, y+r)
	case ToolText:
		return textBounds(s.X1, s.Y1, s.Text, s.Width)
	default:
		return rectFrom(s)
	}
}

// Hit reports whether p is on or inside the stroke.
func (s Stroke) Hit(p Point, slop float64) bool {
	switch s.Tool {
	case ToolPen, ToolMagicErase:
		radius := math.Max(s.Width/2, 4) + slop
		if len(s.Points) == 1 {
			return math.Hypot(p.X-s.Points[0].X, p.Y-s.Points[0].Y) <= radius
		}
		for i := 1; i < len(s.Points); i++ {
			if distToSegment(p, s.Points[i-1], s.Points[i]) <= radius {
				return true
			}
		}
		return false
	case ToolStep:
		x, y := s.X1, s.Y1
		if len(s.Points) > 0 {
			x, y = s.Points[0].X, s.Points[0].Y
		}
		return math.Hypot(p.X-x, p.Y-y) <= stepDiameter(s.Width)/2+slop
	case ToolArrow:
		return hitArrow(s, p, slop)
	case ToolEllipse:
		return hitEllipse(s, p, slop)
	case ToolText:
		return hitText(s, p, slop)
	default:
		r := rectFrom(s)
		r = r.Inset(-int(math.Ceil(slop)))
		return image.Pt(int(math.Round(p.X)), int(math.Round(p.Y))).In(r)
	}
}

func distToSegment(p, a, b Point) float64 {
	dx := b.X - a.X
	dy := b.Y - a.Y
	len2 := dx*dx + dy*dy
	if len2 == 0 {
		return math.Hypot(p.X-a.X, p.Y-a.Y)
	}
	t := ((p.X-a.X)*dx + (p.Y-a.Y)*dy) / len2
	t = clamp(t, 0, 1)
	return math.Hypot(p.X-(a.X+t*dx), p.Y-(a.Y+t*dy))
}

func stepDiameter(width float64) float64 {
	return math.Max(48, width*5.5)
}
