package annotate

import (
	"image"
	"image/color"
	"math"
)

func strokeArrow(dst *image.NRGBA, x1, y1, x2, y2, width float64, ink color.NRGBA) {
	width = math.Max(width, 1)
	stampPath(dst, []Point{{X: x1, Y: y1}, {X: x2, Y: y2}}, width/2, ink)

	angle := math.Atan2(y2-y1, x2-x1)
	head := math.Max(width*4.2, 16)
	left := Point{X: x2 + math.Cos(angle+math.Pi*0.82)*head, Y: y2 + math.Sin(angle+math.Pi*0.82)*head}
	right := Point{X: x2 + math.Cos(angle-math.Pi*0.82)*head, Y: y2 + math.Sin(angle-math.Pi*0.82)*head}
	fillTriangle(dst, Point{X: x2, Y: y2}, left, right, ink)
}

func strokeEllipse(dst *image.NRGBA, x1, y1, x2, y2, width float64, ink color.NRGBA) {
	cx := (x1 + x2) / 2
	cy := (y1 + y2) / 2
	rx := math.Abs(x2-x1) / 2
	ry := math.Abs(y2-y1) / 2
	if rx < 1 {
		rx = 1
	}
	if ry < 1 {
		ry = 1
	}
	radius := math.Max(width/2, 0.8)
	steps := int(math.Ceil(2 * math.Pi * math.Max(rx, ry)))
	if steps < 24 {
		steps = 24
	}
	for i := 0; i < steps; i++ {
		a := 2 * math.Pi * float64(i) / float64(steps)
		stampDisc(dst, cx+rx*math.Cos(a), cy+ry*math.Sin(a), radius, ink)
	}
}

func fillTriangle(dst *image.NRGBA, a, b, c Point, ink color.NRGBA) {
	minX := int(math.Floor(math.Min(a.X, math.Min(b.X, c.X))))
	maxX := int(math.Ceil(math.Max(a.X, math.Max(b.X, c.X))))
	minY := int(math.Floor(math.Min(a.Y, math.Min(b.Y, c.Y))))
	maxY := int(math.Ceil(math.Max(a.Y, math.Max(b.Y, c.Y))))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if !inBounds(dst, x, y) {
				continue
			}
			p := Point{X: float64(x) + 0.5, Y: float64(y) + 0.5}
			if !pointInTriangle(p, a, b, c) {
				continue
			}
			blendAt(dst, x, y, ink)
		}
	}
}

func pointInTriangle(p, a, b, c Point) bool {
	d1 := sign(p, a, b)
	d2 := sign(p, b, c)
	d3 := sign(p, c, a)
	neg := d1 < 0 || d2 < 0 || d3 < 0
	pos := d1 > 0 || d2 > 0 || d3 > 0
	return !(neg && pos)
}

func sign(p, a, b Point) float64 {
	return (p.X-b.X)*(a.Y-b.Y) - (a.X-b.X)*(p.Y-b.Y)
}

func hitArrow(s Stroke, p Point, slop float64) bool {
	radius := math.Max(s.Width/2, 4) + slop
	if distToSegment(p, Point{X: s.X1, Y: s.Y1}, Point{X: s.X2, Y: s.Y2}) <= radius {
		return true
	}
	angle := math.Atan2(s.Y2-s.Y1, s.X2-s.X1)
	head := math.Max(s.Width*4.2, 16) + slop
	left := Point{X: s.X2 + math.Cos(angle+math.Pi*0.82)*head, Y: s.Y2 + math.Sin(angle+math.Pi*0.82)*head}
	right := Point{X: s.X2 + math.Cos(angle-math.Pi*0.82)*head, Y: s.Y2 + math.Sin(angle-math.Pi*0.82)*head}
	return pointInTriangle(p, Point{X: s.X2, Y: s.Y2}, left, right)
}

func hitEllipse(s Stroke, p Point, slop float64) bool {
	cx := (s.X1 + s.X2) / 2
	cy := (s.Y1 + s.Y2) / 2
	rx := math.Abs(s.X2-s.X1)/2 + slop
	ry := math.Abs(s.Y2-s.Y1)/2 + slop
	if rx < 1 || ry < 1 {
		return math.Hypot(p.X-cx, p.Y-cy) <= slop+4
	}
	nx := (p.X - cx) / rx
	ny := (p.Y - cy) / ry
	return nx*nx+ny*ny <= 1
}
