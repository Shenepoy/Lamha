package ui

import (
	"image"
	"math"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/annotate"
	"github.com/lamha-app/lamha/internal/prefs"
)

const (
	defaultMagnifier = 168.0
	minMagnifier     = 88.0
	maxMagnifier     = 320.0
)

func bindCursorTracking(canvas *gtk.DrawingArea, onMove func(x, y float64), onLeave func(), onScroll func(dy float64)) {
	motion := gtk.NewEventControllerMotion()
	motion.ConnectMotion(func(x, y float64) { onMove(x, y) })
	motion.ConnectLeave(func() { onLeave() })
	canvas.AddController(motion)

	scroll := gtk.NewEventControllerScroll(gtk.EventControllerScrollVertical)
	scroll.ConnectScroll(func(dx, dy float64) bool {
		onScroll(dy)
		return true
	})
	canvas.AddController(scroll)
}

func clampMagnifier(size float64) float64 {
	return math.Min(maxMagnifier, math.Max(minMagnifier, size))
}

// magnifierLens is a small overlay that tracks the pointer without redrawing the capture.
type magnifierLens struct {
	layer  *gtk.Fixed
	area   *gtk.DrawingArea
	src    *image.NRGBA
	cx, cy float64
	size   float64
	scale  float64
	placed bool
}

func newMagnifierLens() *magnifierLens {
	lens := &magnifierLens{size: defaultMagnifier, scale: prefs.Current().MagnifierZoom()}
	lens.area = gtk.NewDrawingArea()
	lens.area.AddCSSClass("lamha-lens")
	lens.area.SetCanTarget(false)
	lens.area.SetCanFocus(false)
	lens.resize()
	lens.area.SetDrawFunc(lens.draw)

	lens.layer = gtk.NewFixed()
	lens.layer.AddCSSClass("lamha-lens-layer")
	lens.layer.SetCanTarget(false)
	lens.layer.SetCanFocus(false)
	lens.layer.SetHAlign(gtk.AlignFill)
	lens.layer.SetVAlign(gtk.AlignFill)
	lens.layer.SetHExpand(true)
	lens.layer.SetVExpand(true)
	lens.layer.SetVisible(false)
	return lens
}

func (m *magnifierLens) attach(host *gtk.Overlay) {
	if m == nil || host == nil {
		return
	}
	host.AddOverlay(m.layer)
}

func (m *magnifierLens) setSize(size float64) {
	if m == nil {
		return
	}
	m.size = clampMagnifier(size)
	m.resize()
}

func (m *magnifierLens) resize() {
	side := int(math.Round(m.size))
	m.area.SetSizeRequest(side, side)
	m.area.SetContentWidth(side)
	m.area.SetContentHeight(side)
}

func (m *magnifierLens) hide() {
	if m == nil {
		return
	}
	m.src = nil
	m.layer.SetVisible(false)
}

func (m *magnifierLens) follow(view annotate.View, src *image.NRGBA, widgetX, widgetY float64, areaW, areaH int) {
	if m == nil || src == nil || areaW < 8 || areaH < 8 {
		m.hide()
		return
	}
	m.src = src
	img := view.ToImage(widgetX, widgetY)
	m.cx, m.cy = img.X, img.Y
	zoom := prefs.Current().MagnifierZoom()
	m.scale = view.Scale * zoom
	if m.scale < 0.2 {
		m.scale = zoom
	}

	radius := m.size / 2
	lx := widgetX + radius + 22
	ly := widgetY - 6
	if lx+radius > float64(areaW)-8 {
		lx = widgetX - radius - 22
	}
	if ly-radius < 8 {
		ly = widgetY + radius + 22
	}
	if ly+radius > float64(areaH)-8 {
		ly = widgetY - radius - 22
	}

	x := math.Max(0, math.Round(lx-radius))
	y := math.Max(0, math.Round(ly-radius))
	if !m.placed {
		m.layer.Put(m.area, x, y)
		m.placed = true
	} else {
		m.layer.Move(m.area, x, y)
	}
	m.layer.SetVisible(true)
	m.area.QueueDraw()
}

func (m *magnifierLens) draw(_ *gtk.DrawingArea, cr *cairo.Context, width, height int) {
	if m.src == nil || width < 8 || height < 8 {
		return
	}
	cx, cy := float64(width)/2, float64(height)/2
	radius := math.Min(cx, cy) - 2

	cr.SetOperator(cairo.OperatorClear)
	cr.Paint()
	cr.SetOperator(cairo.OperatorOver)

	cr.Save()
	cr.Arc(cx, cy, radius, 0, 2*math.Pi)
	cr.Clip()

	half := radius/m.scale + 2
	tile, ox, oy := cropAround(m.src, m.cx, m.cy, half)
	if tile != nil {
		cr.Translate(cx, cy)
		cr.Scale(m.scale, m.scale)
		cr.Translate(ox-m.cx, oy-m.cy)
		paintSurface(cr, imageSurface(tile), cairo.FilterNearest)
	}
	cr.Restore()

	cr.SetSourceRGB(1, 1, 1)
	cr.SetLineWidth(2)
	cr.Arc(cx, cy, radius, 0, 2*math.Pi)
	cr.Stroke()
	cr.SetSourceRGBA(0, 0, 0, 0.45)
	cr.SetLineWidth(1)
	cr.Arc(cx, cy, radius+1.5, 0, 2*math.Pi)
	cr.Stroke()

	cr.SetSourceRGB(1, 1, 1)
	cr.SetLineWidth(1)
	cr.MoveTo(cx-6, cy)
	cr.LineTo(cx+6, cy)
	cr.MoveTo(cx, cy-6)
	cr.LineTo(cx, cy+6)
	cr.Stroke()
}

func cropAround(src *image.NRGBA, cx, cy, half float64) (*image.NRGBA, float64, float64) {
	if src == nil {
		return nil, 0, 0
	}
	bounds := src.Bounds()
	minX := int(math.Floor(cx - half))
	minY := int(math.Floor(cy - half))
	maxX := int(math.Ceil(cx + half))
	maxY := int(math.Ceil(cy + half))
	r := image.Rect(minX, minY, maxX, maxY).Intersect(bounds)
	if r.Empty() {
		return nil, 0, 0
	}
	dst := image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	for y := r.Min.Y; y < r.Max.Y; y++ {
		so := src.PixOffset(r.Min.X, y)
		do := dst.PixOffset(0, y-r.Min.Y)
		copy(dst.Pix[do:do+r.Dx()*4], src.Pix[so:so+r.Dx()*4])
	}
	return dst, float64(r.Min.X), float64(r.Min.Y)
}

func drawSelectedStroke(cr *cairo.Context, stroke annotate.Stroke) {
	bounds := stroke.Bounds()
	if bounds.Empty() {
		return
	}
	cr.SetSourceRGB(0.39, 0.73, 1)
	cr.SetLineWidth(1.6)
	cr.SetDash([]float64{6, 4}, 0)
	cr.Rectangle(float64(bounds.Min.X)+0.5, float64(bounds.Min.Y)+0.5, float64(bounds.Dx()-1), float64(bounds.Dy()-1))
	cr.Stroke()
	cr.SetDash(nil, 0)
}

func drawSmoothPath(cr *cairo.Context, points []annotate.Point, width float64) {
	if len(points) == 0 {
		return
	}
	cr.SetLineWidth(width)
	cr.SetLineCap(cairo.LineCapRound)
	cr.SetLineJoin(cairo.LineJoinRound)
	if len(points) == 1 {
		cr.Arc(points[0].X, points[0].Y, width/2, 0, 2*math.Pi)
		cr.Fill()
		return
	}
	cr.MoveTo(points[0].X, points[0].Y)
	for i := 1; i < len(points)-1; i++ {
		midX := (points[i].X + points[i+1].X) / 2
		midY := (points[i].Y + points[i+1].Y) / 2
		cr.CurveTo(points[i].X, points[i].Y, points[i].X, points[i].Y, midX, midY)
	}
	last := points[len(points)-1]
	cr.LineTo(last.X, last.Y)
	cr.Stroke()
}
