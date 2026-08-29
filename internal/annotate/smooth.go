package annotate

import "math"

// SmoothPath turns a raw pointer trail into a denser, rounded polyline.
func SmoothPath(points []Point) []Point {
	points = dedupePoints(points, 0.75)
	if len(points) < 3 {
		return points
	}
	smoothed := chaikin(points)
	smoothed = chaikin(smoothed)
	return densifyPoints(smoothed, 1.8)
}

func dedupePoints(points []Point, minDist float64) []Point {
	if len(points) == 0 {
		return points
	}
	out := []Point{points[0]}
	for _, point := range points[1:] {
		prev := out[len(out)-1]
		if math.Hypot(point.X-prev.X, point.Y-prev.Y) >= minDist {
			out = append(out, point)
		}
	}
	if out[len(out)-1] != points[len(points)-1] {
		out = append(out, points[len(points)-1])
	}
	return out
}

func chaikin(points []Point) []Point {
	if len(points) < 3 {
		return points
	}
	out := make([]Point, 0, len(points)*2)
	out = append(out, points[0])
	for i := 0; i < len(points)-1; i++ {
		a, b := points[i], points[i+1]
		out = append(out,
			Point{X: 0.75*a.X + 0.25*b.X, Y: 0.75*a.Y + 0.25*b.Y},
			Point{X: 0.25*a.X + 0.75*b.X, Y: 0.25*a.Y + 0.75*b.Y},
		)
	}
	out = append(out, points[len(points)-1])
	return out
}

func densifyPoints(points []Point, spacing float64) []Point {
	if spacing < 0.5 || len(points) < 2 {
		return points
	}
	out := []Point{points[0]}
	for i := 1; i < len(points); i++ {
		prev := out[len(out)-1]
		dx := points[i].X - prev.X
		dy := points[i].Y - prev.Y
		dist := math.Hypot(dx, dy)
		steps := int(math.Floor(dist / spacing))
		for s := 1; s <= steps; s++ {
			t := float64(s) * spacing / dist
			out = append(out, Point{X: prev.X + dx*t, Y: prev.Y + dy*t})
		}
		out = append(out, points[i])
	}
	return out
}
