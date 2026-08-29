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
	src := cloneNRGBA(dst)
	bounds := dst.Bounds()
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			var r, g, b, n float64
			add := func(px, py int, weight float64) {
				if weight <= 0 || !image.Pt(px, py).In(bounds) {
					return
				}
				c := src.NRGBAAt(px, py)
				r += float64(c.R) * weight
				g += float64(c.G) * weight
				b += float64(c.B) * weight
				n += weight
			}
			for i := 1; i <= 4; i++ {
				fi := float64(i)
				add(region.Min.X-i, y, 1/math.Max(1, float64(x-region.Min.X)+fi))
				add(region.Max.X-1+i, y, 1/math.Max(1, float64(region.Max.X-1+i-x)))
				add(x, region.Min.Y-i, 1/math.Max(1, float64(y-region.Min.Y)+fi))
				add(x, region.Max.Y-1+i, 1/math.Max(1, float64(region.Max.Y-1+i-y)))
			}
			if n == 0 {
				continue
			}
			dst.SetNRGBA(x, y, color.NRGBA{
				R: uint8(math.Round(r / n)),
				G: uint8(math.Round(g / n)),
				B: uint8(math.Round(b / n)),
				A: 255,
			})
		}
	}
}

func fillMasked(dst *image.NRGBA, mask []bool, w, h, minX, minY, maxX, maxY int, _ float64) {
	if maxX < minX || maxY < minY {
		return
	}
	minX = clampInt(minX, 0, w-1)
	maxX = clampInt(maxX, 0, w-1)
	minY = clampInt(minY, 0, h-1)
	maxY = clampInt(maxY, 0, h-1)

	done := make([]bool, w*h)
	for i, marked := range mask {
		if !marked {
			done[i] = true
		}
	}

	dirs := [...][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {-1, -1}, {-1, 1}, {1, -1}, {1, 1}}
	type pix struct{ x, y int }
	queue := make([]pix, 0, 64)
	queued := make([]bool, w*h)
	enqueue := func(x, y int) {
		if x < minX || y < minY || x > maxX || y > maxY {
			return
		}
		idx := y*w + x
		if !mask[idx] || done[idx] || queued[idx] {
			return
		}
		queued[idx] = true
		queue = append(queue, pix{x, y})
	}
	hasDoneNeighbor := func(x, y int) bool {
		for _, d := range dirs {
			nx, ny := x+d[0], y+d[1]
			if nx >= 0 && ny >= 0 && nx < w && ny < h && done[ny*w+nx] {
				return true
			}
		}
		return false
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if mask[y*w+x] && hasDoneNeighbor(x, y) {
				enqueue(x, y)
			}
		}
	}
	for i := 0; i < len(queue); i++ {
		x, y := queue[i].x, queue[i].y
		idx := y*w + x
		if done[idx] {
			continue
		}
		var r, g, b, n int
		for _, d := range dirs {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || ny < 0 || nx >= w || ny >= h || !done[ny*w+nx] {
				continue
			}
			c := dst.NRGBAAt(nx, ny)
			r += int(c.R)
			g += int(c.G)
			b += int(c.B)
			n++
		}
		if n == 0 {
			continue
		}
		dst.SetNRGBA(x, y, color.NRGBA{R: uint8(r / n), G: uint8(g / n), B: uint8(b / n), A: 255})
		done[idx] = true
		for _, d := range dirs {
			enqueue(x+d[0], y+d[1])
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
