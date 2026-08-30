package ui

import (
	"fmt"
	"image"
	"log"
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type captureTrace struct {
	t0 time.Time
}

func newCaptureTrace() *captureTrace {
	return &captureTrace{t0: time.Now()}
}

func (t *captureTrace) log(stage, format string, args ...any) {
	ms := 0
	if t != nil {
		ms = int(time.Since(t.t0).Milliseconds())
	}
	log.Printf("capture t=%dms %s: %s", ms, stage, fmt.Sprintf(format, args...))
}

func widgetGeom(w gtk.Widgetter) string {
	if w == nil {
		return "nil"
	}
	widget := gtk.BaseWidget(w)
	if widget == nil {
		return "nil"
	}
	return fmt.Sprintf("vis=%v map=%v alloc=%dx%d", widget.Visible(), widget.Mapped(), widget.AllocatedWidth(), widget.AllocatedHeight())
}

func (o *captureOverlay) logChrome(stage string) {
	if o == nil || o.trace == nil {
		return
	}
	if o.window != nil {
		o.trace.log(stage, "window %s fullscreen=%v", widgetGeom(o.window), o.window.IsFullscreen())
	}
	if o.host != nil {
		o.trace.log(stage, "host %s", widgetGeom(o.host))
	}
	if o.canvas != nil {
		o.trace.log(stage, "canvas %s", widgetGeom(o.canvas))
	}
	if o.dock != nil && o.dock.shell != nil {
		o.trace.log(stage, "toolbar %s margin=%d,%d", widgetGeom(o.dock.shell), o.dock.shell.MarginStart(), o.dock.shell.MarginTop())
	}
	if o.statusBox != nil {
		o.trace.log(stage, "status %s", widgetGeom(o.statusBox))
	}
}

// findCenteredBrightSquare reports a bright, roughly square patch near the
// middle of an image. Used to catch the capture flash in screenshots.
func findCenteredBrightSquare(img image.Image, minSide, maxSide int) (image.Rectangle, bool) {
	if img == nil {
		return image.Rectangle{}, false
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w < minSide || h < minSide {
		return image.Rectangle{}, false
	}
	cx, cy := bounds.Min.X+w/2, bounds.Min.Y+h/2
	best := image.Rectangle{}
	bestArea := 0
	for side := minSide; side <= maxSide && side < w && side < h; side += 8 {
		r := image.Rect(cx-side/2, cy-side/2, cx-side/2+side, cy-side/2+side).Intersect(bounds)
		if r.Empty() {
			continue
		}
		if !regionIsBright(img, r, 220) {
			continue
		}
		pad := max(12, side/6)
		outer := image.Rect(r.Min.X-pad, r.Min.Y-pad, r.Max.X+pad, r.Max.Y+pad).Intersect(bounds)
		if regionIsBright(img, outer, 200) {
			continue
		}
		if r.Dx()*r.Dy() > bestArea {
			best = r
			bestArea = r.Dx() * r.Dy()
		}
	}
	return best, bestArea > 0
}

func regionIsBright(img image.Image, r image.Rectangle, minLuma uint32) bool {
	if r.Empty() {
		return false
	}
	var sum uint64
	n := 0
	stepX := max(1, r.Dx()/16)
	stepY := max(1, r.Dy()/16)
	for y := r.Min.Y; y < r.Max.Y; y += stepY {
		for x := r.Min.X; x < r.Max.X; x += stepX {
			cr, cg, cb, ca := img.At(x, y).RGBA()
			if ca < 0x8000 {
				continue
			}
			sum += uint64((cr*299 + cg*587 + cb*114) / 1000 >> 8)
			n++
		}
	}
	if n == 0 {
		return false
	}
	return uint32(sum/uint64(n)) >= minLuma
}

