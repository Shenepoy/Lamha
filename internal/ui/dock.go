package ui

import (
	"log"
	"math"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/graphene"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/i18n"
	"github.com/lamha-app/lamha/internal/prefs"
)

const (
	toolbarMargin = 16
	toolbarSnapPx = 72
)

type floatingToolbar struct {
	host     *gtk.Overlay
	shell    *gtk.Box
	scroll   *gtk.ScrolledWindow
	chrome   *gtk.Box
	tools    *gtk.Box
	sep      *gtk.Separator
	stroke   *strokeControl
	handle   *gtk.DrawingArea
	edge     string
	x          float64
	y          float64
	posX       int
	posY       int
	lastHostW  int
	lastHostH  int
	lastBarW   int
	lastBarH   int
	locked     bool
	snap       bool
	vertical   bool
	dragging   bool
	placed     bool
	grabDX     float64
	grabDY     float64
	lastPX     float64
	lastPY     float64
	onChange   func()
	onSave     func(edge string, x, y float64, vertical bool)
}

func attachFloatingToolbar(host *gtk.Overlay, chrome, tools *gtk.Box, sep *gtk.Separator, stroke *strokeControl) *floatingToolbar {
	store := prefs.Current()
	bar := &floatingToolbar{
		host:     host,
		chrome:   chrome,
		tools:    tools,
		sep:      sep,
		stroke:   stroke,
		edge:     store.ToolbarEdge(),
		x:        store.ToolbarX(),
		y:        store.ToolbarY(),
		vertical: store.ToolbarStacked(),
		locked:   store.ToolbarLocked(),
		snap:     store.ToolbarSnapAlign(),
	}
	if bar.edge == "" {
		bar.edge = prefs.DefaultToolbarEdge
		bar.x = prefs.DefaultToolbarOffset
		bar.y = prefs.DefaultToolbarOffset
	}

	bar.handle = gtk.NewDrawingArea()
	bar.handle.SetCSSClasses([]string{"lamha-grab"})
	unfocusable(&bar.handle.Widget)
	bar.handle.SetDrawFunc(func(_ *gtk.DrawingArea, cr *cairo.Context, width, height int) {
		drawGrabHandle(cr, width, height, bar.vertical)
	})
	bar.updateHandleTip()

	bar.scroll = gtk.NewScrolledWindow()
	bar.scroll.SetHasFrame(false)
	bar.scroll.SetPropagateNaturalWidth(true)
	bar.scroll.SetPropagateNaturalHeight(true)
	bar.scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyNever)
	bar.scroll.SetChild(chrome)

	bar.shell = gtk.NewBox(gtk.OrientationHorizontal, 8)
	bar.shell.SetDirection(gtk.TextDirLTR)
	bar.shell.SetHAlign(gtk.AlignStart)
	bar.shell.SetVAlign(gtk.AlignStart)
	bar.shell.SetHExpand(false)
	bar.shell.SetVExpand(false)
	bar.shell.SetVisible(false)
	bar.shell.Append(bar.handle)
	bar.shell.Append(bar.scroll)

	bar.bindDrag()
	bar.applyOrientation()
	host.AddOverlay(bar.shell)
	bar.shell.ConnectMap(func() {
		bar.relayout(bar.hostSize())
	})
	return bar
}

func (t *floatingToolbar) bindDrag() {
	motion := gtk.NewEventControllerMotion()
	motion.ConnectEnter(func(_, _ float64) {
		if t.locked {
			t.handle.SetCursorFromName("default")
			return
		}
		t.handle.SetCursorFromName("grab")
	})
	t.handle.AddController(motion)

	drag := gtk.NewGestureDrag()
	drag.SetButton(gdk.BUTTON_PRIMARY)
	drag.ConnectDragBegin(func(startX, startY float64) {
		if t.locked {
			return
		}
		px, py, ok := widgetPoint(&t.handle.Widget, t.host, startX, startY)
		if !ok {
			return
		}
		t.dragging = true
		ox, oy := t.origin()
		t.grabDX = px - float64(ox)
		t.grabDY = py - float64(oy)
		t.lastPX, t.lastPY = px, py
		t.handle.SetCursorFromName("grabbing")
		drag.SetState(gtk.EventSequenceClaimed)
	})
	drag.ConnectDragUpdate(func(offsetX, offsetY float64) {
		if !t.dragging {
			return
		}
		startX, startY, ok := drag.StartPoint()
		if !ok {
			return
		}
		px, py, ok := widgetPoint(&t.handle.Widget, t.host, startX+offsetX, startY+offsetY)
		if !ok {
			return
		}
		t.dragTo(px, py)
	})
	drag.ConnectDragEnd(func(_, _ float64) {
		if !t.dragging {
			return
		}
		t.finishDrag()
		t.handle.SetCursorFromName("grab")
	})
	t.handle.AddController(drag)
}

func (t *floatingToolbar) dragTo(pointerX, pointerY float64) {
	hostW, hostH := t.hostSize()
	if hostW < 1 || hostH < 1 {
		return
	}
	t.lastPX, t.lastPY = pointerX, pointerY
	wantVertical := dragToolbarVertical(pointerX, pointerY, float64(hostW), float64(hostH), t.vertical)
	if wantVertical != t.vertical {
		t.vertical = wantVertical
		t.edge = prefs.ToolbarEdgeFloat
		t.applyOrientation()
		t.limitScroll(hostW, hostH)
		t.grabDX, t.grabDY = t.handleHotspot()
	}
	handleW, handleH := t.handleSize()
	t.posX, t.posY = clipDragOrigin(int(math.Round(pointerX-t.grabDX)), int(math.Round(pointerY-t.grabDY)), hostW, hostH, handleW, handleH)
	t.edge = prefs.ToolbarEdgeFloat
	t.placed = true
	t.lastHostW, t.lastHostH = hostW, hostH
	t.place(hostW, hostH)
	if t.onChange != nil {
		t.onChange()
	}
}

func (t *floatingToolbar) handleSize() (w, h int) {
	w, h = 18, 36
	if t.vertical {
		w, h = 32, 16
	}
	if t.handle != nil {
		if aw := t.handle.AllocatedWidth(); aw > 0 {
			w = aw
		}
		if ah := t.handle.AllocatedHeight(); ah > 0 {
			h = ah
		}
	}
	return w, h
}

func (t *floatingToolbar) handleHotspot() (dx, dy float64) {
	w, h := t.handleSize()
	return float64(w) / 2, float64(h) / 2
}

func (t *floatingToolbar) finishDrag() {
	hostW, hostH := t.hostSize()
	if hostW < 1 || hostH < 1 {
		t.dragging = false
		return
	}
	t.edge = snapEdgeFromPointer(t.lastPX, t.lastPY, float64(hostW), float64(hostH), toolbarSnapPx, t.vertical)
	wantVertical := t.vertical
	if !prefs.ToolbarFloating(t.edge) {
		wantVertical = toolbarVertical(t.edge)
	}
	if wantVertical != t.vertical {
		t.vertical = wantVertical
		t.applyOrientation()
		t.limitScroll(hostW, hostH)
		t.grabDX, t.grabDY = t.handleHotspot()
	}
	originX := int(math.Round(t.lastPX - t.grabDX))
	originY := int(math.Round(t.lastPY - t.grabDY))
	t.dragging = false
	t.setPixelOrigin(originX, originY, hostW, hostH)
	if t.snap {
		barW, barH := t.barSize()
		t.x, t.y = snapDockNorms(t.edge, t.x, t.y, hostW, hostH, barW, barH, toolbarMargin, toolbarSnapPx)
	}
	t.applySavedOrigin(hostW, hostH)
	t.place(hostW, hostH)
	if t.onChange != nil {
		t.onChange()
	}
	t.persist()
	t.reseatAfterOrient()
}

func (t *floatingToolbar) reseatAfterOrient() {
	glib.IdleAdd(func() bool {
		if t == nil || t.shell == nil || t.dragging {
			return false
		}
		hostW, hostH := t.hostSize()
		t.applySavedOrigin(hostW, hostH)
		t.place(hostW, hostH)
		if t.onChange != nil {
			t.onChange()
		}
		t.persist()
		return false
	})
}

func (t *floatingToolbar) persist() {
	if t == nil || t.onSave == nil || t.dragging {
		return
	}
	hostW, hostH := t.hostSize()
	barW, barH := t.barSize()
	if !plausibleBarSize(t.vertical, barW, barH, hostW, hostH) {
		return
	}
	t.onSave(t.edge, t.x, t.y, t.vertical)
}

func (t *floatingToolbar) applyOrientation() {
	if !t.dragging && !prefs.ToolbarFloating(t.edge) {
		t.vertical = toolbarVertical(t.edge)
	}
	orient := gtk.OrientationHorizontal
	sepOrient := gtk.OrientationVertical
	classes := []string{"lamha-toolbar", "toolbar", "osd", "lamha-toolbar-horizontal"}
	if t.vertical {
		orient = gtk.OrientationVertical
		sepOrient = gtk.OrientationHorizontal
		classes = []string{"lamha-toolbar", "toolbar", "osd", "lamha-toolbar-vertical"}
	}
	if t.locked {
		classes = append(classes, "lamha-toolbar-locked")
	}
	t.shell.SetOrientation(orient)
	t.chrome.SetOrientation(orient)
	t.tools.SetOrientation(orient)
	if t.vertical {
		t.shell.SetSpacing(4)
		t.chrome.SetSpacing(4)
		t.tools.SetSpacing(0)
	} else {
		t.shell.SetSpacing(8)
		t.chrome.SetSpacing(6)
		t.tools.SetSpacing(2)
	}
	if t.sep != nil {
		t.sep.SetOrientation(sepOrient)
	}
	if t.stroke != nil {
		t.stroke.SetVertical(t.vertical)
	}
	t.shell.SetCSSClasses(classes)
	if t.vertical {
		t.handle.SetSizeRequest(32, 16)
		t.handle.SetContentWidth(32)
		t.handle.SetContentHeight(16)
	} else {
		t.handle.SetSizeRequest(18, 36)
		t.handle.SetContentWidth(18)
		t.handle.SetContentHeight(36)
	}
	t.handle.QueueDraw()
	if t.locked {
		t.handle.SetCursorFromName("default")
	} else if t.dragging {
		t.handle.SetCursorFromName("grabbing")
	} else {
		t.handle.SetCursorFromName("grab")
	}
	t.shell.QueueResize()
}

func (t *floatingToolbar) relayout(hostW, hostH int) {
	if t == nil || t.shell == nil || t.dragging {
		return
	}
	if hostW < 1 || hostH < 1 {
		return
	}
	barW, barH := t.barSize()
	same := t.placed && hostW == t.lastHostW && hostH == t.lastHostH && t.lastBarW == barW && t.lastBarH == barH
	if same {
		return
	}
	t.limitScroll(hostW, hostH)
	t.applySavedOrigin(hostW, hostH)
	t.place(hostW, hostH)
}

func (t *floatingToolbar) applySavedOrigin(hostW, hostH int) {
	if hostW < 1 || hostH < 1 {
		hostW, hostH = t.hostSize()
	}
	if hostW < 1 || hostH < 1 {
		return
	}
	barW, barH := t.barSize()
	t.posX, t.posY = toolbarOrigin(t.edge, t.x, t.y, hostW, hostH, barW, barH, toolbarMargin)
	t.lastHostW, t.lastHostH = hostW, hostH
	if plausibleBarSize(t.vertical, barW, barH, hostW, hostH) {
		t.lastBarW, t.lastBarH = barW, barH
		t.placed = true
	}
}

func (t *floatingToolbar) limitScroll(hostW, hostH int) {
	if t.scroll == nil {
		return
	}
	if hostW < 1 || hostH < 1 {
		hostW, hostH = t.hostSize()
	}
	if t.vertical {
		maxH := hostH - 2*toolbarMargin - 28
		if maxH < 120 {
			maxH = 120
		}
		t.scroll.SetMaxContentHeight(maxH)
		t.scroll.SetMaxContentWidth(-1)
		t.scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
		return
	}
	maxW := hostW - 2*toolbarMargin - 36
	if maxW < 160 {
		maxW = 160
	}
	t.scroll.SetMaxContentWidth(maxW)
	t.scroll.SetMaxContentHeight(-1)
	t.scroll.SetPolicy(gtk.PolicyAutomatic, gtk.PolicyNever)
}

func (t *floatingToolbar) setPixelOrigin(x, y, hostW, hostH int) {
	barW, barH := t.barSize()
	maxX := max(0, hostW-barW)
	maxY := max(0, hostH-barH)
	t.posX = clampInt(x, 0, maxX)
	t.posY = clampInt(y, 0, maxY)
	if !t.dragging {
		switch prefs.NormalizeToolbarEdge(t.edge) {
		case prefs.ToolbarEdgeLeft:
			t.posX = clampInt(toolbarMargin, 0, maxX)
		case prefs.ToolbarEdgeRight:
			t.posX = clampInt(hostW-barW-toolbarMargin, 0, maxX)
		case prefs.ToolbarEdgeTop:
			t.posY = clampInt(toolbarMargin, 0, maxY)
		case prefs.ToolbarEdgeBottom:
			t.posY = clampInt(hostH-barH-toolbarMargin, 0, maxY)
		}
	}
	t.syncNorms(hostW, hostH, barW, barH)
	t.lastHostW, t.lastHostH = hostW, hostH
	if plausibleBarSize(t.vertical, barW, barH, hostW, hostH) {
		t.lastBarW, t.lastBarH = barW, barH
	}
	t.placed = true
}

func (t *floatingToolbar) syncNorms(hostW, hostH, barW, barH int) {
	t.x, t.y = normalizeOrigin(t.edge, float64(t.posX), float64(t.posY), float64(hostW), float64(hostH), float64(barW), float64(barH), toolbarMargin)
}

func (t *floatingToolbar) place(hostW, hostH int) {
	if hostW < 1 || hostH < 1 {
		hostW, hostH = t.hostSize()
	}
	if hostW < 200 || hostH < 200 {
		return
	}
	barW, barH := t.barSize()
	if (barW < 40 || barH < 40) && t.shell.Visible() {
		return
	}
	x, y := t.posX, t.posY
	if t.dragging {
		handleW, handleH := t.handleSize()
		x, y = clipDragOrigin(x, y, hostW, hostH, handleW, handleH)
	} else {
		maxX := max(0, hostW-barW)
		maxY := max(0, hostH-barH)
		x, y = clampInt(t.posX, 0, maxX), clampInt(t.posY, 0, maxY)
	}
	if t.shell.MarginStart() != x || t.shell.MarginTop() != y {
		t.shell.SetMarginStart(x)
		t.shell.SetMarginEnd(0)
		t.shell.SetMarginTop(y)
		t.shell.SetMarginBottom(0)
	}
	if !t.shell.Visible() {
		t.shell.SetVisible(true)
		log.Printf("capture toolbar shown alloc=%dx%d margin=%d,%d", t.shell.AllocatedWidth(), t.shell.AllocatedHeight(), x, y)
	}
}

func (t *floatingToolbar) hostSize() (int, int) {
	if t == nil || t.host == nil {
		return 0, 0
	}
	return t.host.AllocatedWidth(), t.host.AllocatedHeight()
}

func (t *floatingToolbar) barSize() (int, int) {
	hostW, hostH := t.hostSize()
	w, h := t.measuredBarSize(hostW, hostH)
	w, h = sizeForToolbarOrientation(t.vertical, w, h)
	if hostW > 0 && w > hostW {
		w = hostW
	}
	if hostH > 0 && h > hostH {
		h = hostH
	}
	return w, h
}

func (t *floatingToolbar) measuredBarSize(hostW, hostH int) (int, int) {
	if w, h, ok := t.chromeBarSize(hostW, hostH); ok {
		return w, h
	}
	if t.shell != nil {
		aw, ah := t.shell.AllocatedWidth(), t.shell.AllocatedHeight()
		if plausibleBarSize(t.vertical, aw, ah, hostW, hostH) {
			return aw, ah
		}
	}
	if t.lastBarW >= 40 && t.lastBarH >= 24 {
		return t.lastBarW, t.lastBarH
	}
	return estimateBarSize(t.vertical, hostW, hostH)
}

func (t *floatingToolbar) chromeBarSize(hostW, hostH int) (int, int, bool) {
	if t.chrome == nil {
		return 0, 0, false
	}
	hw, hh := t.handleSize()
	cw, ch := widgetNaturalSize(&t.chrome.Widget)
	if aw := t.chrome.AllocatedWidth(); aw > cw {
		cw = aw
	}
	if ah := t.chrome.AllocatedHeight(); ah > ch {
		ch = ah
	}
	space := 8
	if t.vertical {
		space = 4
	}
	var w, h int
	if t.vertical {
		w = max(hw, cw)
		h = hh + space + ch
	} else {
		w = hw + space + cw
		h = max(hh, ch)
	}
	if !plausibleBarSize(t.vertical, w, h, hostW, hostH) {
		return 0, 0, false
	}
	return w, h, true
}

func widgetNaturalSize(widget *gtk.Widget) (int, int) {
	if widget == nil {
		return 0, 0
	}
	_, mw, _, _ := widget.Measure(gtk.OrientationHorizontal, -1)
	_, mh, _, _ := widget.Measure(gtk.OrientationVertical, -1)
	return mw, mh
}

func plausibleBarSize(vertical bool, w, h, hostW, hostH int) bool {
	if w < 40 || h < 24 {
		return false
	}
	if vertical {
		return (hostW <= 0 || w <= hostW/2) && (hostH <= 0 || h < hostH)
	}
	return (hostH <= 0 || h <= hostH/2) && (hostW <= 0 || w < hostW)
}

func estimateBarSize(vertical bool, hostW, hostH int) (int, int) {
	if vertical {
		h := 420
		if hostH > 0 {
			h = min(h, max(160, hostH-2*toolbarMargin))
		}
		return 56, h
	}
	w := 560
	if hostW > 0 {
		w = min(w, max(200, hostW-2*toolbarMargin))
	}
	return w, 56
}

func sizeForToolbarOrientation(vertical bool, w, h int) (int, int) {
	if w < 1 || h < 1 {
		return w, h
	}
	if vertical && w > h {
		return h, w
	}
	if !vertical && h > w {
		return h, w
	}
	return w, h
}

func (t *floatingToolbar) origin() (int, int) {
	return t.shell.MarginStart(), t.shell.MarginTop()
}

func (t *floatingToolbar) updateHandleTip() {
	if t == nil || t.handle == nil {
		return
	}
	if t.locked {
		t.handle.SetTooltipText(i18n.T("Toolbar position is locked. Unlock it in Settings to move it."))
		return
	}
		t.handle.SetTooltipText(i18n.T("Drag to move the toolbar. Drop near an edge to snap."))
}

func widgetPoint(src *gtk.Widget, target gtk.Widgetter, x, y float64) (float64, float64, bool) {
	pt := graphene.NewPointAlloc()
	pt.Init(float32(x), float32(y))
	out, ok := src.ComputePoint(target, pt)
	if !ok || out == nil {
		return 0, 0, false
	}
	return float64(out.X()), float64(out.Y()), true
}

func toolbarVertical(edge string) bool {
	return prefs.ToolbarVertical(edge)
}

func clipDragOrigin(x, y, hostW, hostH, handleW, handleH int) (int, int) {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if maxX := hostW - handleW; handleW > 0 && x > maxX {
		x = max(0, maxX)
	}
	if maxY := hostH - handleH; handleH > 0 && y > maxY {
		y = max(0, maxY)
	}
	return x, y
}

func dragToolbarVertical(px, py, hostW, hostH float64, current bool) bool {
	if hostW < 1 {
		hostW = 1
	}
	if hostH < 1 {
		hostH = 1
	}
	side := math.Min(px, hostW-px)
	along := math.Min(py, hostH-py)
	const touch = 2.0
	if side > touch && along > touch {
		return current
	}
	if current {
		return along >= side
	}
	return side < along
}

func nearestToolbarEdge(x, y, w, h float64) string {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	edges := []struct {
		edge string
		dist float64
	}{
		{prefs.ToolbarEdgeTop, y},
		{prefs.ToolbarEdgeBottom, h - y},
		{prefs.ToolbarEdgeLeft, x},
		{prefs.ToolbarEdgeRight, w - x},
	}
	best := edges[0]
	for _, item := range edges[1:] {
		if item.dist < best.dist {
			best = item
		}
	}
	return best.edge
}

func snapEdgeFromPointer(px, py, hostW, hostH, snapPx float64, preferVertical bool) string {
	if hostW < 1 {
		hostW = 1
	}
	if hostH < 1 {
		hostH = 1
	}
	along := []struct {
		edge string
		dist float64
	}{
		{prefs.ToolbarEdgeTop, py},
		{prefs.ToolbarEdgeBottom, hostH - py},
	}
	across := []struct {
		edge string
		dist float64
	}{
		{prefs.ToolbarEdgeLeft, px},
		{prefs.ToolbarEdgeRight, hostW - px},
	}
	first, second := along, across
	if preferVertical {
		first, second = across, along
	}
	if edge, ok := closestEdgeWithin(first, snapPx); ok {
		return edge
	}
	if edge, ok := closestEdgeWithin(second, snapPx); ok {
		return edge
	}
	return prefs.ToolbarEdgeFloat
}

func closestEdgeWithin(edges []struct {
	edge string
	dist float64
}, snapPx float64) (string, bool) {
	if len(edges) == 0 {
		return prefs.ToolbarEdgeFloat, false
	}
	best := edges[0]
	for _, item := range edges[1:] {
		if item.dist < best.dist {
			best = item
		}
	}
	if best.dist <= snapPx {
		return best.edge, true
	}
	return prefs.ToolbarEdgeFloat, false
}

func toolbarOrigin(edge string, xNorm, yNorm float64, hostW, hostH, barW, barH, margin int) (x, y int) {
	cx, cy := placementCenter(edge, xNorm, yNorm, float64(hostW), float64(hostH), float64(barW), float64(barH), float64(margin))
	x = int(math.Round(cx - float64(barW)/2))
	y = int(math.Round(cy - float64(barH)/2))
	maxX := max(0, hostW-barW)
	maxY := max(0, hostH-barH)
	x = clampInt(x, 0, maxX)
	y = clampInt(y, 0, maxY)
	if margin < 0 {
		margin = 0
	}
	switch prefs.NormalizeToolbarEdge(edge) {
	case prefs.ToolbarEdgeBottom:
		return x, clampInt(hostH-barH-margin, 0, maxY)
	case prefs.ToolbarEdgeLeft:
		return clampInt(margin, 0, maxX), y
	case prefs.ToolbarEdgeRight:
		return clampInt(hostW-barW-margin, 0, maxX), y
	case prefs.ToolbarEdgeFloat:
		return x, y
	default:
		return x, clampInt(margin, 0, maxY)
	}
}

// placementCenter is the stored location: 0.5 is the toolbar's visual center
// on the screen (or in the middle of a snapped edge).
func placementCenter(edge string, xNorm, yNorm, hostW, hostH, barW, barH, margin float64) (cx, cy float64) {
	xNorm = prefs.ClampToolbarOffset(xNorm)
	yNorm = prefs.ClampToolbarOffset(yNorm)
	if margin < 0 {
		margin = 0
	}
	switch prefs.NormalizeToolbarEdge(edge) {
	case prefs.ToolbarEdgeLeft:
		return margin + barW/2, alongCenter(yNorm, hostH, barH, margin)
	case prefs.ToolbarEdgeRight:
		return hostW - margin - barW/2, alongCenter(yNorm, hostH, barH, margin)
	case prefs.ToolbarEdgeBottom:
		return alongCenter(xNorm, hostW, barW, margin), hostH - margin - barH/2
	case prefs.ToolbarEdgeFloat:
		return alongCenter(xNorm, hostW, barW, 0), alongCenter(yNorm, hostH, barH, 0)
	default:
		return alongCenter(xNorm, hostW, barW, margin), margin + barH/2
	}
}

func alongCenter(norm, host, bar, margin float64) float64 {
	lo := margin + bar/2
	hi := host - margin - bar/2
	if hi < lo {
		return (lo + hi) / 2
	}
	return lo + norm*(hi-lo)
}

func normalizeOrigin(edge string, originX, originY, hostW, hostH, barW, barH, margin float64) (x, y float64) {
	cx := originX + barW/2
	cy := originY + barH/2
	if prefs.ToolbarFloating(edge) {
		return prefs.ClampToolbarOffset(normAlong(cx, hostW, barW, 0)), prefs.ClampToolbarOffset(normAlong(cy, hostH, barH, 0))
	}
	if toolbarVertical(edge) {
		return prefs.DefaultToolbarOffset, prefs.ClampToolbarOffset(normAlong(cy, hostH, barH, margin))
	}
	return prefs.ClampToolbarOffset(normAlong(cx, hostW, barW, margin)), prefs.DefaultToolbarOffset
}

func normAlong(center, host, bar, margin float64) float64 {
	lo := margin + bar/2
	hi := host - margin - bar/2
	if hi-lo <= 1 {
		return prefs.DefaultToolbarOffset
	}
	return (center - lo) / (hi - lo)
}

func clampFloat(v, lo, hi float64) float64 {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (t *floatingToolbar) coversBottom(hostH int) bool {
	if t == nil {
		return false
	}
	if t.edge == prefs.ToolbarEdgeBottom {
		return true
	}
	_, y := t.origin()
	_, h := t.barSize()
	return y+h > hostH-80
}

func snapDockNorms(edge string, x, y float64, hostW, hostH, barW, barH, margin int, snapPx float64) (float64, float64) {
	if prefs.ToolbarFloating(edge) {
		travelX := float64(max(1, hostW-barW))
		travelY := float64(max(1, hostH-barH))
		return snapToolbarOffset(x, travelX, snapPx), snapToolbarOffset(y, travelY, snapPx)
	}
	if toolbarVertical(edge) {
		travel := float64(hostH - barH - 2*margin)
		return prefs.DefaultToolbarOffset, snapToolbarOffset(y, travel, snapPx)
	}
	travel := float64(hostW - barW - 2*margin)
	return snapToolbarOffset(x, travel, snapPx), prefs.DefaultToolbarOffset
}

func snapToolbarOffset(offset, travel, snapPx float64) float64 {
	offset = prefs.ClampToolbarOffset(offset)
	if travel <= 0 || snapPx <= 0 {
		return offset
	}
	threshold := snapPx / travel
	if threshold > 0.45 {
		threshold = 0.45
	}
	for _, target := range []float64{0, 0.5, 1} {
		if math.Abs(offset-target) <= threshold {
			return target
		}
	}
	return offset
}

func clampInt(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func drawGrabHandle(cr *cairo.Context, width, height int, verticalBar bool) {
	if width < 1 || height < 1 {
		return
	}
	cr.SetSourceRGBA(1, 1, 1, 0.55)
	cols, rows := 2, 3
	if verticalBar {
		cols, rows = 3, 2
	}
	padX := float64(width) * 0.28
	padY := float64(height) * 0.22
	spanX := float64(width) - 2*padX
	spanY := float64(height) - 2*padY
	radius := math.Min(2.1, math.Min(spanX, spanY)/6)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			x := padX
			y := padY
			if cols > 1 {
				x += spanX * float64(col) / float64(cols-1)
			}
			if rows > 1 {
				y += spanY * float64(row) / float64(rows-1)
			}
			cr.Arc(x, y, radius, 0, 2*math.Pi)
			cr.Fill()
		}
	}
}
