package ui

import (
	"image"
	"math"

	"github.com/diamondburned/gotk4/pkg/cairo"

	"github.com/lamha-app/lamha/internal/annotate"
)

type selHandle int

const (
	selHandleNone selHandle = iota
	selHandleMove
	selHandleN
	selHandleS
	selHandleE
	selHandleW
	selHandleNW
	selHandleNE
	selHandleSW
	selHandleSE
)

const (
	selectionBorderScreen = 3.5
	selectionAccentScreen = 2.0
	selectionDotScreen    = 5.5
	selectionHitScreen    = 16.0
	minSelectionPx        = 8
)

type selBox struct {
	minX, minY, maxX, maxY float64
}

func selBoxFromRect(r image.Rectangle) selBox {
	if r.Empty() {
		return selBox{}
	}
	return selBox{float64(r.Min.X), float64(r.Min.Y), float64(r.Max.X), float64(r.Max.Y)}
}

func (b selBox) empty() bool {
	return b.maxX-b.minX < 1 || b.maxY-b.minY < 1
}

func (b selBox) rect() image.Rectangle {
	if b.empty() {
		return image.Rectangle{}
	}
	return image.Rect(int(math.Round(b.minX)), int(math.Round(b.minY)), int(math.Round(b.maxX)), int(math.Round(b.maxY)))
}

func (b selBox) width() float64  { return b.maxX - b.minX }
func (b selBox) height() float64 { return b.maxY - b.minY }

func selectionHit(sel image.Rectangle, p annotate.Point, scale float64) selHandle {
	return selectionHitBox(selBoxFromRect(sel), p, scale)
}

func selectionHitBox(sel selBox, p annotate.Point, scale float64) selHandle {
	if sel.empty() {
		return selHandleNone
	}
	if scale < 0.05 {
		scale = 0.05
	}
	pad := selectionHitScreen / scale
	x, y := p.X, p.Y
	if x < sel.minX-pad || x > sel.maxX+pad || y < sel.minY-pad || y > sel.maxY+pad {
		return selHandleNone
	}
	nearL := math.Abs(x-sel.minX) <= pad
	nearR := math.Abs(x-sel.maxX) <= pad
	nearT := math.Abs(y-sel.minY) <= pad
	nearB := math.Abs(y-sel.maxY) <= pad
	switch {
	case nearT && nearL:
		return selHandleNW
	case nearT && nearR:
		return selHandleNE
	case nearB && nearL:
		return selHandleSW
	case nearB && nearR:
		return selHandleSE
	case nearT:
		return selHandleN
	case nearB:
		return selHandleS
	case nearL:
		return selHandleW
	case nearR:
		return selHandleE
	case x >= sel.minX && x <= sel.maxX && y >= sel.minY && y <= sel.maxY:
		return selHandleMove
	default:
		return selHandleNone
	}
}

func resizeSelection(orig image.Rectangle, handle selHandle, p annotate.Point, bounds image.Rectangle) image.Rectangle {
	return resizeSelBox(selBoxFromRect(orig), handle, p, bounds).rect()
}

func resizeSelBox(orig selBox, handle selHandle, p annotate.Point, bounds image.Rectangle) selBox {
	if handle == selHandleNone || handle == selHandleMove || orig.empty() {
		return orig
	}
	minW := float64(minSelectionPx)
	left, right, top, bot := orig.minX, orig.maxX, orig.minY, orig.maxY
	x, y := p.X, p.Y
	switch handle {
	case selHandleN:
		top = math.Min(y, bot-minW)
	case selHandleS:
		bot = math.Max(y, top+minW)
	case selHandleW:
		left = math.Min(x, right-minW)
	case selHandleE:
		right = math.Max(x, left+minW)
	case selHandleNW:
		left = math.Min(x, right-minW)
		top = math.Min(y, bot-minW)
	case selHandleNE:
		right = math.Max(x, left+minW)
		top = math.Min(y, bot-minW)
	case selHandleSW:
		left = math.Min(x, right-minW)
		bot = math.Max(y, top+minW)
	case selHandleSE:
		right = math.Max(x, left+minW)
		bot = math.Max(y, top+minW)
	}
	return clampSelBox(selBox{left, top, right, bot}, bounds)
}

func moveSelection(orig image.Rectangle, dx, dy float64, bounds image.Rectangle) image.Rectangle {
	return moveSelBox(selBoxFromRect(orig), dx, dy, bounds).rect()
}

func moveSelBox(orig selBox, dx, dy float64, bounds image.Rectangle) selBox {
	if orig.empty() {
		return orig
	}
	w, h := orig.width(), orig.height()
	x := orig.minX + dx
	y := orig.minY + dy
	minX, minY := float64(bounds.Min.X), float64(bounds.Min.Y)
	maxX, maxY := float64(bounds.Max.X), float64(bounds.Max.Y)
	if x < minX {
		x = minX
	}
	if y < minY {
		y = minY
	}
	if x+w > maxX {
		x = maxX - w
	}
	if y+h > maxY {
		y = maxY - h
	}
	if x < minX {
		x = minX
	}
	if y < minY {
		y = minY
	}
	return selBox{x, y, x + w, y + h}
}

func clampSelBox(b selBox, bounds image.Rectangle) selBox {
	if b.maxX < b.minX {
		b.minX, b.maxX = b.maxX, b.minX
	}
	if b.maxY < b.minY {
		b.minY, b.maxY = b.maxY, b.minY
	}
	minX, minY := float64(bounds.Min.X), float64(bounds.Min.Y)
	maxX, maxY := float64(bounds.Max.X), float64(bounds.Max.Y)
	if b.minX < minX {
		b.minX = minX
	}
	if b.minY < minY {
		b.minY = minY
	}
	if b.maxX > maxX {
		b.maxX = maxX
	}
	if b.maxY > maxY {
		b.maxY = maxY
	}
	if b.maxX-b.minX < float64(minSelectionPx) && b.minX+float64(minSelectionPx) <= maxX {
		b.maxX = b.minX + float64(minSelectionPx)
	}
	if b.maxY-b.minY < float64(minSelectionPx) && b.minY+float64(minSelectionPx) <= maxY {
		b.maxY = b.minY + float64(minSelectionPx)
	}
	return b
}

func selBoxFromPoints(x1, y1, x2, y2 float64) selBox {
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	return selBox{x1, y1, x2, y2}
}

func selectionCursor(handle selHandle) string {
	switch handle {
	case selHandleN:
		return "n-resize"
	case selHandleS:
		return "s-resize"
	case selHandleE:
		return "e-resize"
	case selHandleW:
		return "w-resize"
	case selHandleNW:
		return "nw-resize"
	case selHandleNE:
		return "ne-resize"
	case selHandleSW:
		return "sw-resize"
	case selHandleSE:
		return "se-resize"
	case selHandleMove:
		return "grab"
	default:
		return ""
	}
}

func drawSelectionMask(cr *cairo.Context, size image.Point, selection image.Rectangle) {
	drawSelectionBox(cr, size, selBoxFromRect(selection))
}

func drawSelectionBox(cr *cairo.Context, size image.Point, sel selBox) {
	if sel.empty() {
		cr.SetSourceRGBA(0, 0, 0, 0.35)
		cr.Rectangle(0, 0, float64(size.X), float64(size.Y))
		cr.Fill()
		return
	}
	sel = clampSelBox(sel, image.Rect(0, 0, size.X, size.Y))
	if sel.empty() {
		return
	}
	cr.SetSourceRGBA(0, 0, 0, 0.5)
	cr.Rectangle(0, 0, float64(size.X), sel.minY)
	cr.Rectangle(0, sel.minY, sel.minX, sel.height())
	cr.Rectangle(sel.maxX, sel.minY, float64(size.X)-sel.maxX, sel.height())
	cr.Rectangle(0, sel.maxY, float64(size.X), float64(size.Y)-sel.maxY)
	cr.Fill()

	scale := maxFloat(crScale(cr), 0.5)
	ar, ag, ab := desktopAccentRGB()
	cr.SetAntialias(cairo.AntialiasDefault)
	cr.SetLineCap(cairo.LineCapRound)
	cr.SetLineJoin(cairo.LineJoinRound)
	cr.Rectangle(sel.minX, sel.minY, sel.width(), sel.height())
	cr.SetSourceRGBA(0, 0, 0, 0.55)
	cr.SetLineWidth((selectionBorderScreen + selectionAccentScreen + 2.5) / scale)
	cr.StrokePreserve()
	cr.SetSourceRGB(1, 1, 1)
	cr.SetLineWidth((selectionBorderScreen + selectionAccentScreen) / scale)
	cr.StrokePreserve()
	cr.SetSourceRGB(ar, ag, ab)
	cr.SetLineWidth(selectionAccentScreen / scale)
	cr.Stroke()

	drawSelectionDots(cr, sel, scale, ar, ag, ab)
}

func drawSelectionDots(cr *cairo.Context, sel selBox, scale, ar, ag, ab float64) {
	if sel.width() < float64(minSelectionPx) || sel.height() < float64(minSelectionPx) {
		return
	}
	radius := selectionDotScreen / scale
	points := [][2]float64{
		{sel.minX, sel.minY},
		{sel.maxX, sel.minY},
		{sel.minX, sel.maxY},
		{sel.maxX, sel.maxY},
		{sel.minX + sel.width()/2, sel.minY},
		{sel.minX + sel.width()/2, sel.maxY},
		{sel.minX, sel.minY + sel.height()/2},
		{sel.maxX, sel.minY + sel.height()/2},
	}
	for _, pt := range points {
		drawSelectionDot(cr, pt[0], pt[1], radius, scale, ar, ag, ab)
	}
}

func drawSelectionDot(cr *cairo.Context, x, y, radius, scale, ar, ag, ab float64) {
	cr.NewPath()
	cr.Arc(x, y, radius+2.4/scale, 0, 2*math.Pi)
	cr.SetSourceRGBA(0, 0, 0, 0.5)
	cr.Fill()
	cr.Arc(x, y, radius+1.2/scale, 0, 2*math.Pi)
	cr.SetSourceRGB(1, 1, 1)
	cr.Fill()
	cr.Arc(x, y, radius, 0, 2*math.Pi)
	cr.SetSourceRGB(ar, ag, ab)
	cr.Fill()
	cr.Arc(x-radius*0.28, y-radius*0.28, radius*0.32, 0, 2*math.Pi)
	cr.SetSourceRGBA(1, 1, 1, 0.4)
	cr.Fill()
}
