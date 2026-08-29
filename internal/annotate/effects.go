package annotate

import (
	"image"
	"image/color"
	"math"
)

func boxBlur(dst *image.NRGBA, region image.Rectangle, radius int) {
	region = region.Intersect(dst.Bounds())
	if region.Empty() || radius < 1 {
		return
	}
	src := cloneNRGBA(dst)
	tmp := image.NewNRGBA(dst.Rect)
	copy(tmp.Pix, dst.Pix)

	extent := radius
	// Horizontal pass into tmp, vertical pass back into dst.
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			var r, g, b, n int
			for xx := x - extent; xx <= x+extent; xx++ {
				px := clampInt(xx, region.Min.X, region.Max.X-1)
				c := src.NRGBAAt(px, y)
				r += int(c.R)
				g += int(c.G)
				b += int(c.B)
				n++
			}
			tmp.SetNRGBA(x, y, color.NRGBA{R: uint8(r / n), G: uint8(g / n), B: uint8(b / n), A: 255})
		}
	}
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			var r, g, b, n int
			for yy := y - extent; yy <= y+extent; yy++ {
				py := clampInt(yy, region.Min.Y, region.Max.Y-1)
				c := tmp.NRGBAAt(x, py)
				r += int(c.R)
				g += int(c.G)
				b += int(c.B)
				n++
			}
			dst.SetNRGBA(x, y, color.NRGBA{R: uint8(r / n), G: uint8(g / n), B: uint8(b / n), A: 255})
		}
	}
}

func magicErase(dst *image.NRGBA, points []Point, width float64) {
	radius := math.Max(width/2, 4)
	bounds := dst.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	mask := make([]bool, w*h)
	minX, minY := w, h
	maxX, maxY := -1, -1
	mark := func(x, y int) {
		if x < 0 || y < 0 || x >= w || y >= h {
			return
		}
		mask[y*w+x] = true
		if x < minX {
			minX = x
		}
		if y < minY {
			minY = y
		}
		if x > maxX {
			maxX = x
		}
		if y > maxY {
			maxY = y
		}
	}
	stampMask(points, radius, mark)
	if maxX < minX {
		return
	}

	fillMasked(dst, mask, w, h, minX, minY, maxX, maxY, math.Max(10, radius*2))
}

func magicEraseRect(dst *image.NRGBA, region image.Rectangle) {
	region = region.Intersect(dst.Bounds())
	if region.Empty() {
		return
	}
	bounds := dst.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	mask := make([]bool, w*h)
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			mask[y*w+x] = true
		}
	}
	fillMasked(dst, mask, w, h, region.Min.X, region.Min.Y, region.Max.X-1, region.Max.Y-1, math.Max(12, float64(max(region.Dx(), region.Dy()))*0.08))
}

func fillMasked(dst *image.NRGBA, mask []bool, w, h, minX, minY, maxX, maxY int, windowF float64) {
	src := cloneNRGBA(dst)
	window := int(math.Max(10, windowF))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if !mask[y*w+x] {
				continue
			}
			var r, g, b, n int
			for yy := y - window; yy <= y+window; yy++ {
				for xx := x - window; xx <= x+window; xx++ {
					if xx < 0 || yy < 0 || xx >= w || yy >= h || mask[yy*w+xx] {
						continue
					}
					c := src.NRGBAAt(xx, yy)
					r += int(c.R)
					g += int(c.G)
					b += int(c.B)
					n++
				}
			}
			if n == 0 {
				continue
			}
			dst.SetNRGBA(x, y, color.NRGBA{R: uint8(r / n), G: uint8(g / n), B: uint8(b / n), A: 255})
		}
	}
}

func stampMask(points []Point, radius float64, mark func(x, y int)) {
	r := int(math.Ceil(radius))
	r2 := radius * radius
	var prev Point
	for i, point := range points {
		if i > 0 {
			dx := point.X - prev.X
			dy := point.Y - prev.Y
			dist := math.Hypot(dx, dy)
			steps := int(math.Ceil(dist / math.Max(radius*0.4, 1)))
			for s := 1; s <= steps; s++ {
				t := float64(s) / float64(steps)
				stampMaskDisc(prev.X+dx*t, prev.Y+dy*t, r, r2, mark)
			}
		} else {
			stampMaskDisc(point.X, point.Y, r, r2, mark)
		}
		prev = point
	}
}

func stampMaskDisc(cx, cy float64, r int, r2 float64, mark func(x, y int)) {
	ix := int(math.Round(cx))
	iy := int(math.Round(cy))
	for y := iy - r; y <= iy+r; y++ {
		for x := ix - r; x <= ix+r; x++ {
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			if dx*dx+dy*dy <= r2 {
				mark(x, y)
			}
		}
	}
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
