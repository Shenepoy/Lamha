package ui

import (
	"image"
	"log"
	"os"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/annotate"
	"github.com/lamha-app/lamha/internal/brand"
	"github.com/lamha-app/lamha/internal/grab"
	"github.com/lamha-app/lamha/internal/i18n"
	"github.com/lamha-app/lamha/internal/keys"
	"github.com/lamha-app/lamha/internal/prefs"
)

type captureOverlay struct {
	parent        *Window
	window        *gtk.Window
	canvas        *gtk.DrawingArea
	status        *gtk.Label
	copyOnSave    *gtk.ToggleButton
	undoButton    *gtk.Button
	redoButton    *gtk.Button
	stroke        *strokeControl
	tools         *toolSwitcher
	colors        *colorSwitcher
	doc           *annotate.Document
	staging       string
	mode          CaptureMode
	surface       *cairo.Surface
	tool          annotate.Tool
	color         annotate.Color
	width         float64
	draft         *annotate.Stroke
	view          annotate.View
	selection     image.Rectangle
	selBox        selBox
	selStart      annotate.Point
	selOrig       selBox
	selHandle     selHandle
	selecting     bool
	selected      int
	moving        bool
	moveLast      annotate.Point
	cursorX       float64
	cursorY       float64
	cursorIn      bool
	magnifierSize float64
	fading        bool
	host          *gtk.Overlay
	text          textInput
	redraw        redrawPump
	lens          *magnifierLens
	dock          *floatingToolbar
	statusBox     *gtk.Box
	frames        []image.Rectangle
	hovered       image.Rectangle
	windowLocked  bool
	borrowed      bool
	savedChild    gtk.Widgetter
	savedTitlebar gtk.Widgetter
	savedTitle    string
	wasMaximized  bool
	keysCtl       *gtk.EventControllerKey
	trace         *captureTrace
	loggedDraw    bool
	stepPop       *stepNumberPop
	justDragged   bool
}

func (w *Window) openCaptureOverlay(mode CaptureMode, staging string, frames []grab.WindowFrame) {
	w.openCaptureOverlayOn(mode, staging, frames, monitorRect(w.window))
}

func (w *Window) openCaptureOverlayOn(mode CaptureMode, staging string, frames []grab.WindowFrame, mon image.Rectangle) {
	if w.overlay != nil {
		w.overlay.close(false)
	}

	doc, err := annotate.Open(staging)
	if err != nil {
		os.Remove(staging)
		w.failCapture(err)
		return
	}

	ov := &captureOverlay{
		parent:        w,
		doc:           doc,
		staging:       staging,
		mode:          mode,
		tool:          annotate.ToolSelect,
		color:         editorColors[0].color,
		width:         8,
		selected:      -1,
		magnifierSize: defaultMagnifier,
		trace:         w.captureTrace,
	}
	if ov.trace == nil {
		ov.trace = newCaptureTrace()
	}
	ov.trace.log("overlay", "open mode=%d staging=%s monitor=%v", mode, staging, mon)
	switch mode {
	case CaptureScreen:
		size := doc.Size()
		ov.setSelection(image.Rect(0, 0, size.X, size.Y))
	case CaptureWindow:
		if mon.Empty() {
			mon = monitorRect(w.window)
		}
		ov.frames = mapWindowFrames(frames, doc.Size(), mon)
	}
	w.parkTransientWindows()
	ov.build()
	ov.refreshSurface()
	w.overlay = ov
	ov.trace.log("overlay", "surface ready borrowed=%v", ov.borrowed)

	width, height, monitor := displayBounds(w.window)
	if !mon.Empty() {
		width, height = mon.Dx(), mon.Dy()
	}
	ov.trace.log("overlay", "show %dx%d monitor=%v", width, height, monitor != nil)
	ov.showOnMonitor(width, height, monitor)
	ov.logChrome("shown")
}

func (o *captureOverlay) build() {
	ensureToolbarCSS()
	if o.parent.captureWindowPlan.dedicatedOverlay {
		o.window = gtk.NewWindow()
		if app := o.parent.window.Application(); app != nil {
			o.window.SetApplication(app)
		}
		o.window.SetTitle(i18n.T("Lamha capture"))
		o.window.SetIconName(brand.Name)
		o.window.SetDecorated(false)
		o.window.ConnectCloseRequest(func() bool {
			o.close(true)
			return true
		})
		o.trace.log("build", "create dedicated capture window")
	} else {
		o.window = &o.parent.window.Window
		o.borrowed = true
		o.savedChild = o.window.Child()
		o.savedTitlebar = o.window.Titlebar()
		o.savedTitle = o.window.Title()
		o.wasMaximized = o.window.IsMaximized()
		o.window.SetTitle(i18n.T("Lamha capture"))
		if o.savedTitlebar != nil {
			o.window.SetTitlebar(nil)
		}
		o.trace.log("build", "borrow main window mapped=%v visible=%v child=%v titlebar=%v",
			o.window.Mapped(), o.window.Visible(), o.savedChild != nil, o.savedTitlebar != nil)
	}
	o.window.AddCSSClass("lamha-capture")

	keysCtl := gtk.NewEventControllerKey()
	o.keysCtl = keysCtl
	keysCtl.SetPropagationPhase(gtk.PhaseCapture)
	keysCtl.SetPropagationLimit(gtk.LimitNone)
	keysCtl.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		if o.text.active() {
			if keyval == gdk.KEY_Escape {
				o.text.cancel()
				o.refreshSurface()
				o.status.SetText(o.hint())
				o.canvas.QueueDraw()
				return true
			}
			return false
		}
		return handleAnnotateKey(keyval, state, o.keyActions())
	})
	o.window.AddController(keysCtl)

	o.canvas = gtk.NewDrawingArea()
	o.canvas.AddCSSClass("lamha-capture-canvas")
	o.canvas.SetHExpand(true)
	o.canvas.SetVExpand(true)
	o.canvas.SetFocusable(true)
	o.canvas.SetCursorFromName(o.cursorName())
	o.canvas.SetDrawFunc(o.draw)
	o.stepPop = newStepNumberPop(o.nudgeStep)
	o.bindGestures()
	bindCursorTracking(o.canvas, func(x, y float64) {
		o.cursorX, o.cursorY = x, y
		o.cursorIn = true
		if o.mode == CaptureWindow && !o.windowLocked {
			next := grab.FrameAt(o.frames, image.Pt(int(o.view.ToImage(x, y).X), int(o.view.ToImage(x, y).Y)))
			if !next.Eq(o.hovered) {
				o.hovered = next
				o.canvas.QueueDraw()
			}
		}
		o.updateLens()
		o.updateSelectionCursor()
	}, func() {
		o.cursorIn = false
		if !o.hovered.Empty() {
			o.hovered = image.Rectangle{}
			o.canvas.QueueDraw()
		}
		o.updateLens()
		o.updateSelectionCursor()
	})

	o.host = gtk.NewOverlay()
	o.host.AddCSSClass("lamha-capture-host")
	o.host.SetDirection(gtk.TextDirLTR)
	o.host.SetChild(o.canvas)
	o.lens = newMagnifierLens()
	o.lens.setSize(o.magnifierSize)
	o.lens.attach(o.host)
	o.buildChrome()
	o.canvas.ConnectResize(func(width, height int) {
		o.trace.log("resize", "canvas=%dx%d", width, height)
		if o.dock != nil {
			o.dock.relayout(width, height)
		}
		o.syncStatusPlacement()
		o.logChrome("resize")
	})

	o.window.SetChild(o.host)
}

func (o *captureOverlay) buildChrome() {
	ensureToolbarCSS()

	chrome := gtk.NewBox(gtk.OrientationHorizontal, 6)
	chrome.SetHAlign(gtk.AlignCenter)
	chrome.SetVAlign(gtk.AlignCenter)
	chrome.SetHExpand(false)
	if i18n.RTL() {
		chrome.SetDirection(gtk.TextDirRTL)
	}

	tools := gtk.NewBox(gtk.OrientationHorizontal, 2)
	tools.SetVAlign(gtk.AlignCenter)
	tools.SetHAlign(gtk.AlignCenter)
	o.tools = appendToolToggles(tools, captureTools(), o.tool, iconInkOnDark, o.setTool)
	chrome.Append(tools)

	sep := gtk.NewSeparator(gtk.OrientationVertical)
	chrome.Append(sep)

	o.colors = appendColorSwatches(chrome, 0, o.setColorIndex)

	o.stroke = newStrokeControl(o.width, true, true, strokeWidthTip(), o.setWidth)
	chrome.Append(o.stroke.box)

	o.undoButton = newIconButton("edit-undo-symbolic", withKey(i18n.T("Undo"), keys.Undo))
	o.undoButton.SetSensitive(false)
	o.undoButton.ConnectClicked(o.undo)
	chrome.Append(o.undoButton)

	o.redoButton = newIconButton("edit-redo-symbolic", withKey(i18n.T("Redo"), keys.Redo))
	o.redoButton.SetSensitive(false)
	o.redoButton.ConnectClicked(o.redo)
	chrome.Append(o.redoButton)

	o.copyOnSave = newIconToggle("edit-copy-symbolic", i18n.T("Also copy to clipboard when saving"))
	o.copyOnSave.SetActive(true)
	chrome.Append(o.copyOnSave)

	done := newIconButton("document-save-symbolic", withKey(i18n.T("Save the selected area"), keys.Confirm))
	done.SetCSSClasses([]string{"suggested-action"})
	done.ConnectClicked(o.confirm)
	chrome.Append(done)

	cancel := newIconButton("window-close-symbolic", withKey(i18n.T("Discard this capture"), keys.Escape))
	cancel.ConnectClicked(func() { o.close(true) })
	chrome.Append(cancel)

	o.status = gtk.NewLabel(o.hint())
	o.status.SetHAlign(gtk.AlignCenter)
	o.status.SetWrap(true)
	o.status.SetCSSClasses([]string{"dim-label", "dimmed"})
	if i18n.RTL() {
		o.status.SetDirection(gtk.TextDirRTL)
	}

	o.statusBox = gtk.NewBox(gtk.OrientationHorizontal, 0)
	o.statusBox.SetCSSClasses([]string{"osd", "lamha-capture-status"})
	o.statusBox.SetHAlign(gtk.AlignCenter)
	o.statusBox.SetVisible(false)
	o.statusBox.Append(o.status)
	o.host.AddOverlay(o.statusBox)

	o.dock = attachFloatingToolbar(o.host, chrome, tools, sep, o.stroke)
	o.dock.onChange = o.syncStatusPlacement
	o.dock.onSave = func(edge string, x, y float64, vertical bool) {
		if err := prefs.Current().SetToolbarDock(edge, x, y, vertical); err != nil {
			o.status.SetText(i18n.Tf("Could not save settings: %v", err))
		}
	}
	o.syncStatusPlacement()
}

func (o *captureOverlay) syncStatusPlacement() {
	if o.statusBox == nil {
		return
	}
	if o.host != nil && (o.host.AllocatedWidth() < 200 || o.host.AllocatedHeight() < 200) {
		o.statusBox.SetVisible(false)
		return
	}
	if !o.statusBox.Visible() {
		o.statusBox.SetVisible(true)
	}
	if o.dock != nil && o.dock.coversBottom(o.host.AllocatedHeight()) {
		o.statusBox.SetVAlign(gtk.AlignStart)
		o.statusBox.SetMarginTop(toolbarMargin)
		o.statusBox.SetMarginBottom(0)
		return
	}
	o.statusBox.SetVAlign(gtk.AlignEnd)
	o.statusBox.SetMarginTop(0)
	o.statusBox.SetMarginBottom(toolbarMargin)
}

func (o *captureOverlay) hint() string {
	if o.mode == CaptureWindow {
		if o.windowLocked || (!o.selection.Empty() && len(o.frames) <= 1) {
			return i18n.T("Window captured. Save or mark it up.")
		}
		if len(o.frames) == 0 {
			return i18n.T("Could not list windows. Drag to select a window region.")
		}
		if o.selection.Empty() {
			return i18n.T("Click a window to select it. Save captures that window.")
		}
		return i18n.T("Window selected. Save to capture it, or click another.")
	}
	if o.mode == CaptureArea && o.selection.Empty() {
		return i18n.T("Drag to select. Shortcuts are editable in the main window.")
	}
	if o.mode == CaptureArea {
		return i18n.T("Drag a corner or edge to resize, or inside to move.") + " " + withKey(i18n.T("Save"), keys.Confirm) + "."
	}
	return withKey(i18n.T("Click a mark to edit it."), keys.Delete) + " " + i18n.T("Double-click text to edit.") + " " + withKey(i18n.T("Undo"), keys.Undo) + ". " + withKey(i18n.T("Save"), keys.Confirm) + "."
}

func (o *captureOverlay) bindGestures() {
	right := gtk.NewGestureClick()
	right.SetButton(gdk.BUTTON_SECONDARY)
	right.SetPropagationPhase(gtk.PhaseCapture)
	right.ConnectPressed(func(nPress int, x, y float64) {
		o.rightClickCancel()
	})
	o.canvas.AddController(right)

	click := gtk.NewGestureClick()
	click.SetButton(gdk.BUTTON_PRIMARY)
	click.ConnectPressed(func(nPress int, x, y float64) {
		if nPress >= 2 && o.beginTextEdit(o.view.ToImage(x, y)) {
			click.SetState(gtk.EventSequenceClaimed)
		}
	})
	click.ConnectReleased(func(nPress int, x, y float64) {
		if nPress < 1 || o.text.active() {
			return
		}
		point := o.view.ToImage(x, y)
		if o.tool == annotate.ToolSelect || o.tool == annotate.ToolMove {
			if o.justDragged {
				o.justDragged = false
				o.maybeShowStepPop(o.selected)
				o.refreshActions()
				o.canvas.QueueDraw()
				return
			}
			o.selected = o.doc.Hit(point)
			if stroke, ok := o.doc.Stroke(o.selected); ok {
				o.adoptStroke(stroke)
			}
			o.maybeShowStepPop(o.selected)
			o.refreshActions()
			o.canvas.QueueDraw()
			return
		}
		if o.tool == annotate.ToolText {
			o.commitTyping()
			o.text.begin(o.host, o.view, point, o.color, o.width, func() {
				o.commitTyping()
				o.refreshSurface()
				o.canvas.QueueDraw()
				o.status.SetText(o.hint())
			})
			o.status.SetText(i18n.T("Type, then Enter to place. Escape cancels."))
			o.updateLens()
			o.canvas.QueueDraw()
			return
		}
		if o.tool != annotate.ToolStep {
			return
		}
		o.selected = -1
		o.doc.Add(annotate.Stroke{
			Tool:   annotate.ToolStep,
			Color:  o.color,
			Width:  o.width,
			X1:     point.X,
			Y1:     point.Y,
			Points: []annotate.Point{point},
			Step:   o.doc.NextStep(),
		})
		o.refreshSurface()
		o.canvas.QueueDraw()
	})
	o.canvas.AddController(click)

	drag := gtk.NewGestureDrag()
	drag.SetButton(gdk.BUTTON_PRIMARY)
	drag.ConnectDragBegin(func(startX, startY float64) {
		if o.text.active() {
			return
		}
		start := o.view.ToImage(startX, startY)
		o.hideStepPop()
		if o.tool == annotate.ToolSelect || o.tool == annotate.ToolMove {
			if hit := o.doc.Hit(start); hit >= 0 {
				o.selected = hit
				if stroke, ok := o.doc.Stroke(hit); ok {
					o.adoptStroke(stroke)
				}
				o.moving = true
				o.moveLast = start
				o.selecting = false
				o.draft = nil
				o.beginLiveMove()
				o.canvas.SetCursorFromName("grabbing")
				o.canvas.QueueDraw()
				return
			}
			if o.mode == CaptureWindow && o.tool == annotate.ToolSelect && !o.windowLocked {
				if frame := grab.FrameAt(o.frames, image.Pt(int(start.X), int(start.Y))); !frame.Empty() {
					o.lockWindow(frame)
					return
				}
			}
			if o.tool == annotate.ToolMove {
				if o.selected >= 0 {
					o.moving = true
					o.moveLast = start
					o.beginLiveMove()
					o.canvas.SetCursorFromName("grabbing")
				}
				o.draft = nil
				o.canvas.QueueDraw()
				return
			}
			if o.canResizeSelection() {
				if handle := selectionHitBox(o.liveSel(), start, o.view.Scale); handle != selHandleNone {
					o.selected = -1
					o.selecting = false
					o.selHandle = handle
					o.selOrig = o.liveSel()
					o.selStart = start
					o.draft = nil
					if handle == selHandleMove {
						o.canvas.SetCursorFromName("grabbing")
					} else {
						o.canvas.SetCursorFromName(selectionCursor(handle))
					}
					o.canvas.QueueDraw()
					return
				}
			}
			o.selected = -1
			o.selHandle = selHandleNone
			o.selecting = true
			o.selStart = start
			o.draft = nil
			o.canvas.QueueDraw()
			return
		}
		if toolPlacesOnClick(o.tool) {
			return
		}
		o.selected = -1
		o.selecting = false
		o.draft = &annotate.Stroke{
			Tool:   o.tool,
			Color:  o.color,
			Width:  o.width,
			X1:     start.X,
			Y1:     start.Y,
			X2:     start.X,
			Y2:     start.Y,
			Points: []annotate.Point{start},
		}
		o.canvas.QueueDraw()
	})
	drag.ConnectDragUpdate(func(offsetX, offsetY float64) {
		startX, startY, ok := drag.StartPoint()
		if !ok {
			return
		}
		point := o.view.ToImage(startX+offsetX, startY+offsetY)
		if o.moving && o.selected >= 0 {
			o.doc.Move(o.selected, point.X-o.moveLast.X, point.Y-o.moveLast.Y)
			o.moveLast = point
			o.queueFrame()
			return
		}
		if o.selHandle != selHandleNone {
			bounds := image.Rect(0, 0, o.doc.Size().X, o.doc.Size().Y)
			if o.selHandle == selHandleMove {
				o.setSelBox(moveSelBox(o.selOrig, point.X-o.selStart.X, point.Y-o.selStart.Y, bounds))
			} else {
				o.setSelBox(resizeSelBox(o.selOrig, o.selHandle, point, bounds))
			}
			o.queueSelection()
			return
		}
		if o.selecting {
			o.setSelBox(selBoxFromPoints(o.selStart.X, o.selStart.Y, point.X, point.Y))
			o.queueSelection()
			return
		}
		if o.draft == nil {
			return
		}
		o.draft.X2 = point.X
		o.draft.Y2 = point.Y
		if o.draft.Tool == annotate.ToolPen || o.draft.Tool == annotate.ToolMagicErase {
			o.draft.Points = append(o.draft.Points, point)
		}
		o.queueFrame()
	})
	drag.ConnectDragEnd(func(offsetX, offsetY float64) {
		if o.moving {
			o.moving = false
			o.justDragged = true
			o.refreshSurface()
			o.maybeShowStepPop(o.selected)
			o.canvas.SetCursorFromName(o.cursorName())
			o.canvas.QueueDraw()
			o.status.SetText(i18n.Tf("Moved the selected mark. Arrow keys nudge. %s.", withKey(i18n.T("Duplicate"), keys.Duplicate)))
			return
		}
		if o.selHandle != selHandleNone {
			o.selHandle = selHandleNone
			if o.selection.Dx() < minSelectionPx || o.selection.Dy() < minSelectionPx {
				o.setSelection(image.Rectangle{})
			}
			o.status.SetText(o.hint())
			o.updateSelectionCursor()
			o.canvas.QueueDraw()
			return
		}
		if o.selecting {
			o.selecting = false
			if o.selection.Dx() < minSelectionPx || o.selection.Dy() < minSelectionPx {
				o.setSelection(image.Rectangle{})
			} else if o.mode == CaptureWindow && !o.windowLocked {
				o.lockWindow(o.selection)
				return
			}
			o.status.SetText(o.hint())
			o.updateSelectionCursor()
			o.canvas.QueueDraw()
			return
		}
		if o.draft == nil {
			return
		}
		stroke := *o.draft
		o.draft = nil
		if !strokeReady(stroke) {
			return
		}
		o.doc.Add(stroke)
		o.refreshSurface()
		o.canvas.QueueDraw()
	})
	o.canvas.AddController(drag)
}

func (o *captureOverlay) rightClickCancel() {
	o.hideStepPop()
	if o.draft != nil || o.selecting || o.selHandle != selHandleNone {
		o.draft = nil
		o.selecting = false
		o.selHandle = selHandleNone
		if o.mode == CaptureArea {
			o.setSelection(image.Rectangle{})
		}
		o.status.SetText(o.hint())
		o.updateSelectionCursor()
		o.canvas.QueueDraw()
		return
	}
	if !o.selection.Empty() {
		o.setSelection(image.Rectangle{})
		o.status.SetText(o.hint())
		o.updateSelectionCursor()
		o.canvas.QueueDraw()
		return
	}
	o.close(true)
}

func (o *captureOverlay) draw(_ *gtk.DrawingArea, cr *cairo.Context, width, height int) {
	if !o.loggedDraw {
		o.loggedDraw = true
		o.trace.log("draw", "first canvas=%dx%d surface=%v", width, height, o.surface != nil)
		o.logChrome("first-draw")
	}
	size := o.doc.Size()
	o.view = annotate.Fit(size.X, size.Y, width, height)

	if o.mode == CaptureWindow {
		cr.SetSourceRGB(0, 0, 0)
	} else {
		cr.SetSourceRGB(0.05, 0.05, 0.06)
	}
	cr.Paint()
	if o.surface == nil {
		return
	}

	cr.Save()
	cr.Translate(o.view.OffsetX, o.view.OffsetY)
	cr.Scale(o.view.Scale, o.view.Scale)
	paintSurface(cr, o.surface, cairo.FilterNearest)
	if o.mode == CaptureWindow && !o.windowLocked && !o.hovered.Empty() && o.hovered != o.selection {
		drawWindowHover(cr, o.hovered)
	}
	drawSelectionBox(cr, size, o.liveSel())
	if stroke, ok := o.doc.Stroke(o.selected); ok {
		if o.moving {
			drawDraft(cr, stroke)
		}
		drawSelectedStroke(cr, stroke)
	}
	if o.text.active() && o.text.entry == nil && o.text.draft.text != "" {
		drawDraft(cr, o.text.draft.stroke())
	}
	if o.draft != nil {
		drawDraft(cr, *o.draft)
	}
	cr.Restore()
}

func (o *captureOverlay) canResizeSelection() bool {
	if o == nil || o.tool != annotate.ToolSelect || o.selection.Empty() {
		return false
	}
	return o.mode != CaptureWindow || o.windowLocked
}

func (o *captureOverlay) updateSelectionCursor() {
	if o == nil || o.canvas == nil {
		return
	}
	if o.selecting || o.selHandle != selHandleNone || o.moving {
		return
	}
	if !o.cursorIn || !o.canResizeSelection() {
		o.canvas.SetCursorFromName(o.cursorName())
		return
	}
	point := o.view.ToImage(o.cursorX, o.cursorY)
	if name := selectionCursor(selectionHitBox(o.liveSel(), point, o.view.Scale)); name != "" {
		o.canvas.SetCursorFromName(name)
		return
	}
	o.canvas.SetCursorFromName(o.cursorName())
}

func crScale(cr *cairo.Context) float64 {
	matrix := cr.Matrix()
	if matrix == nil {
		return 1
	}
	return matrix.Xx
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func (o *captureOverlay) queueFrame() {
	o.redraw.request(func() {
		if o.canvas != nil {
			o.canvas.QueueDraw()
		}
	})
}

func (o *captureOverlay) queueSelection() {
	if o != nil && o.canvas != nil {
		o.canvas.QueueDraw()
	}
}

func (o *captureOverlay) liveSel() selBox {
	if o == nil {
		return selBox{}
	}
	if !o.selBox.empty() {
		return o.selBox
	}
	return selBoxFromRect(o.selection)
}

func (o *captureOverlay) setSelection(r image.Rectangle) {
	if o == nil {
		return
	}
	o.selection = r
	o.selBox = selBoxFromRect(r)
}

func (o *captureOverlay) setSelBox(b selBox) {
	if o == nil {
		return
	}
	if b.empty() {
		o.selection = image.Rectangle{}
		o.selBox = selBox{}
		return
	}
	o.selBox = b
	o.selection = b.rect()
}

func (o *captureOverlay) beginLiveMove() {
	if o.selected < 0 {
		return
	}
	o.surface = imageSurface(o.doc.RenderExcept(o.selected))
	o.updateLens()
}

func (o *captureOverlay) skipStroke() int {
	if o.moving && o.selected >= 0 {
		return o.selected
	}
	if o.text.active() && o.text.replace >= 0 {
		return o.text.replace
	}
	return -1
}

func (o *captureOverlay) refreshSurface() {
	if skip := o.skipStroke(); skip >= 0 {
		o.surface = imageSurface(o.doc.RenderExcept(skip))
	} else {
		o.surface = imageSurface(o.doc.Raster())
	}
	o.refreshActions()
	o.updateLens()
}

func (o *captureOverlay) updateLens() {
	if o.lens == nil || o.canvas == nil || o.doc == nil {
		return
	}
	if !o.cursorIn || o.moving || o.selecting || o.selHandle != selHandleNone || o.draft != nil || o.text.active() {
		o.lens.hide()
		return
	}
	if o.canvas.AllocatedWidth() < 200 || o.canvas.AllocatedHeight() < 200 {
		o.lens.hide()
		return
	}
	size := o.doc.Size()
	o.view = annotate.Fit(size.X, size.Y, o.canvas.AllocatedWidth(), o.canvas.AllocatedHeight())
	o.lens.follow(o.view, o.doc.Raster(), o.cursorX, o.cursorY, o.canvas.AllocatedWidth(), o.canvas.AllocatedHeight())
}

func (o *captureOverlay) beginTextEdit(point annotate.Point) bool {
	if o.tool != annotate.ToolMove && o.tool != annotate.ToolSelect {
		return false
	}
	hit := o.doc.Hit(point)
	stroke, ok := o.doc.Stroke(hit)
	if !ok || stroke.Tool != annotate.ToolText {
		return false
	}
	if o.moving {
		o.moving = false
	}
	o.commitTyping()
	o.selected = hit
	o.color = stroke.Color
	o.adoptStroke(stroke)
	o.text.beginReplace(o.host, o.view, stroke, hit, func() {
		o.selected = applyTextCommit(o.doc, &o.text)
		o.refreshSurface()
		o.canvas.QueueDraw()
		o.status.SetText(o.hint())
	})
	o.refreshSurface()
	o.canvas.QueueDraw()
	o.status.SetText(i18n.T("Edit text, then Enter to save. Escape cancels."))
	return true
}

func (o *captureOverlay) refreshActions() {
	if o.undoButton != nil {
		o.undoButton.SetSensitive(o.doc.CanUndo())
	}
	if o.redoButton != nil {
		o.redoButton.SetSensitive(o.doc.CanRedo())
	}
}

func (o *captureOverlay) setTool(tool annotate.Tool) {
	if o.tool != tool {
		o.commitTyping()
	}
	o.tool = tool
	o.draft = nil
	if tool != annotate.ToolSelect && tool != annotate.ToolMove {
		o.hideStepPop()
	}
	if o.tools != nil {
		o.tools.activate(tool)
	}
	if o.canvas != nil {
		o.updateSelectionCursor()
		o.canvas.QueueDraw()
	}
}

func (o *captureOverlay) setColorIndex(index int) {
	if index < 0 || index >= len(editorColors) {
		return
	}
	o.color = editorColors[index].color
	if o.colors != nil {
		o.colors.activate(index)
	}
	if o.text.active() {
		o.text.draft.color = o.color
		o.canvas.QueueDraw()
	}
	if stroke, ok := o.doc.Stroke(o.selected); ok {
		stroke.Color = o.color
		o.doc.Replace(o.selected, stroke)
		o.refreshSurface()
		o.canvas.QueueDraw()
	}
}

func (o *captureOverlay) adoptStroke(stroke annotate.Stroke) {
	o.width = clampStroke(stroke.Width)
	o.stroke.SetValue(o.width)
}

func (o *captureOverlay) setWidth(width float64) {
	width = clampStroke(width)
	o.width = width
	o.stroke.SetValue(width)
	if o.text.active() {
		o.text.draft.width = width
		o.canvas.QueueDraw()
	}
	if stroke, ok := o.doc.Stroke(o.selected); ok {
		stroke.Width = width
		o.doc.Replace(o.selected, stroke)
		o.refreshSurface()
		o.canvas.QueueDraw()
	}
}

func (o *captureOverlay) keyActions() keyActions {
	return keyActions{
		setTool:    o.setTool,
		setColor:   o.setColorIndex,
		nudgeWidth: func(delta float64) { o.setWidth(o.width + delta) },
		nudge:      o.nudgeSelected,
		undo:       o.undo,
		redo:       o.redo,
		delete:     o.deleteSelected,
		duplicate:  o.duplicateSelected,
		save:       o.confirm,
		toggleCopy: func() {
			if o.copyOnSave != nil {
				o.copyOnSave.SetActive(!o.copyOnSave.Active())
			}
		},
		escape: func() bool {
			if o.text.active() {
				o.text.cancel()
				o.refreshSurface()
				o.status.SetText(o.hint())
				o.canvas.QueueDraw()
				return true
			}
			if o.selected >= 0 {
				o.selected = -1
				o.hideStepPop()
				o.canvas.QueueDraw()
				return true
			}
			o.rightClickCancel()
			return true
		},
	}
}

func (o *captureOverlay) nudgeSelected(dx, dy float64) bool {
	if o.selected < 0 {
		return false
	}
	if !o.doc.Move(o.selected, dx, dy) {
		return false
	}
	o.refreshSurface()
	o.maybeShowStepPop(o.selected)
	o.canvas.QueueDraw()
	return true
}

func (o *captureOverlay) duplicateSelected() {
	next := o.doc.Duplicate(o.selected)
	if next < 0 {
		return
	}
	o.selected = next
	o.hideStepPop()
	o.refreshSurface()
	o.canvas.QueueDraw()
	o.status.SetText(i18n.T("Duplicated the selected mark."))
}

func (o *captureOverlay) commitTyping() {
	if !o.text.active() {
		return
	}
	o.selected = applyTextCommit(o.doc, &o.text)
	o.refreshSurface()
}

func (o *captureOverlay) deleteSelected() {
	if !o.doc.Remove(o.selected) {
		return
	}
	o.selected = -1
	o.hideStepPop()
	o.refreshSurface()
	o.canvas.QueueDraw()
	o.status.SetText(i18n.T("Removed the selected mark."))
}

func (o *captureOverlay) undo() {
	if o.draft != nil {
		o.draft = nil
		o.canvas.QueueDraw()
		return
	}
	if !o.doc.Undo() {
		return
	}
	o.selected = -1
	o.hideStepPop()
	o.refreshSurface()
	o.canvas.QueueDraw()
	o.status.SetText(i18n.T("Undid the last mark."))
}

func (o *captureOverlay) redo() {
	if !o.doc.Redo() {
		return
	}
	o.selected = -1
	o.hideStepPop()
	o.refreshSurface()
	o.canvas.QueueDraw()
	o.status.SetText(i18n.T("Redid the last mark."))
}

func (o *captureOverlay) nudgeStep(index, delta int) {
	next := o.doc.AdjustStep(index, delta, prefs.Current().RenumberSteps(), prefs.Current().DeleteZeroSteps())
	o.selected = next
	o.refreshSurface()
	o.refreshActions()
	if next < 0 {
		o.hideStepPop()
		o.canvas.QueueDraw()
		return
	}
	o.maybeShowStepPop(next)
	o.canvas.QueueDraw()
}

func (o *captureOverlay) maybeShowStepPop(index int) {
	stroke, ok := o.doc.Stroke(index)
	if !ok || stroke.Tool != annotate.ToolStep {
		o.hideStepPop()
		return
	}
	if o.stepPop == nil {
		o.stepPop = newStepNumberPop(o.nudgeStep)
	}
	o.stepPop.present(o.canvas, o.view, stroke, index)
}

func (o *captureOverlay) hideStepPop() {
	if o.stepPop != nil {
		o.stepPop.hide()
	}
}

func (o *captureOverlay) confirm() {
	o.commitTyping()
	rendered := o.doc.RenderRegion(o.selection)
	copyOnSave := o.copyOnSave.Active()
	parent := o.parent
	o.dismiss(func() {
		parent.setBusy(true, i18n.T("Saving capture…"))
		if parent.restoreAfterCapture {
			parent.Present()
		}
	})

	go func() {
		saved, err := parent.store.SaveImage(rendered)
		glib.IdleAdd(func() {
			parent.setBusy(false, "")
			if err != nil {
				message := i18n.Tf("Could not save: %s", err.Error())
				parent.status.SetText(message)
				if !parent.restoreAfterCapture {
					parent.Notify("Lamha", message)
				}
				return
			}

			message := i18n.Tf("Saved %s", saved.Path)
			if copyOnSave {
				if err := copyImageFile(saved.Path); err != nil {
					message = i18n.Tf("Saved, but clipboard copy failed: %s", err.Error())
				} else {
					message = i18n.T("Saved and copied to the clipboard.")
				}
			}
			parent.showCapture(saved, message)
			parent.prependHistory(saved)
			if !parent.restoreAfterCapture {
				parent.Notify("Lamha", message)
			}
		})
	}()
}

func (o *captureOverlay) close(cancelled bool) {
	o.dismiss(func() {
		if cancelled {
			o.parent.status.SetText(i18n.T("Capture cancelled."))
			if o.parent.restoreAfterCapture {
				o.parent.Present()
			}
		}
	})
}

func (o *captureOverlay) dismiss(after func()) {
	if o.fading {
		return
	}
	o.text.cancel()
	o.hideStepPop()
	if o.lens != nil {
		o.lens.hide()
	}
	o.fading = true
	if o.parent.overlay == o {
		o.parent.overlay = nil
	}
	if o.staging != "" {
		os.Remove(o.staging)
		o.staging = ""
	}
	win := o.window
	o.window = nil
	if win == nil {
		if after != nil {
			after()
		}
		return
	}
	o.releaseWindow(win)
	if after != nil {
		after()
	}
	if o.parent != nil {
		o.parent.restoreParkedWindows()
	}
}

func (o *captureOverlay) releaseWindow(win *gtk.Window) {
	if o.keysCtl != nil {
		win.RemoveController(o.keysCtl)
		o.keysCtl = nil
	}
	win.RemoveCSSClass("lamha-capture")
	if o.borrowed {
		o.trace.log("release", "restore main window titlebar=%v child=%v maximized=%v",
			o.savedTitlebar != nil, o.savedChild != nil, o.wasMaximized)
		win.Unfullscreen()
		if !o.wasMaximized {
			win.Unmaximize()
		}
		if o.savedTitle != "" {
			win.SetTitle(o.savedTitle)
		}
		if o.savedTitlebar != nil {
			win.SetTitlebar(o.savedTitlebar)
		}
		if o.savedChild != nil {
			win.SetChild(o.savedChild)
		}
		if o.parent != nil && !o.parent.restoreAfterCapture {
			win.SetVisible(false)
		}
		return
	}
	win.SetVisible(false)
	win.Destroy()
}

func (o *captureOverlay) lockWindow(frame image.Rectangle) {
	if o.windowLocked || o.staging == "" || frame.Empty() {
		o.setSelection(frame)
		o.hovered = frame
		o.selecting = false
		o.draft = nil
		o.status.SetText(o.hint())
		o.canvas.QueueDraw()
		return
	}
	if err := grab.CropPNG(o.staging, frame); err != nil {
		log.Printf("window crop: %v", err)
		o.setSelection(frame)
		o.hovered = frame
		o.selecting = false
		o.draft = nil
		o.status.SetText(o.hint())
		o.canvas.QueueDraw()
		return
	}
	doc, err := annotate.Open(o.staging)
	if err != nil {
		log.Printf("window crop reload: %v", err)
		o.setSelection(frame)
		o.status.SetText(o.hint())
		o.canvas.QueueDraw()
		return
	}
	o.doc = doc
	o.frames = nil
	o.hovered = image.Rectangle{}
	o.windowLocked = true
	o.selecting = false
	o.draft = nil
	size := doc.Size()
	o.setSelection(image.Rect(0, 0, size.X, size.Y))
	o.refreshSurface()
	o.canvas.SetCursorFromName(o.cursorName())
	o.status.SetText(o.hint())
	o.canvas.QueueDraw()
}

func (o *captureOverlay) showOnMonitor(width, height int, monitor *gdk.Monitor) {
	o.coverMonitor(width, height)
	if monitor != nil {
		o.window.FullscreenOnMonitor(monitor)
	} else {
		o.window.Fullscreen()
	}
	clearOpaqueRegion(o.window)
	o.trace.log("show", "present borrowed=%v mapped=%v fullscreen=%v",
		o.borrowed, o.window.Mapped(), o.window.IsFullscreen())
	o.window.Present()
	if o.canvas != nil {
		o.canvas.GrabFocus()
		o.canvas.QueueDraw()
	}
	glib.TimeoutAdd(80, func() bool {
		if o.window == nil {
			return false
		}
		clearOpaqueRegion(o.window)
		if monitor != nil && !o.window.IsFullscreen() {
			o.trace.log("show", "retry fullscreen")
			o.window.FullscreenOnMonitor(monitor)
		}
		if o.canvas != nil {
			o.canvas.QueueDraw()
		}
		o.logChrome("show+80ms")
		return false
	})
}

func clearOpaqueRegion(win *gtk.Window) {
	if win == nil {
		return
	}
	surfacer := win.Surface()
	if surfacer == nil {
		return
	}
	surface := gdk.BaseSurface(surfacer)
	if surface == nil {
		return
	}
	region, err := cairo.RegionCreate()
	if err != nil || region == nil {
		return
	}
	surface.SetOpaqueRegion(region)
}

func (o *captureOverlay) coverMonitor(width, height int) {
	if width < 1 || height < 1 {
		size := o.doc.Size()
		width, height = size.X, size.Y
	}
	if width < 1 {
		width, height = 1920, 1080
	}
	if !o.borrowed {
		o.window.SetDefaultSize(width, height)
	}
	if o.canvas != nil {
		o.canvas.SetContentWidth(width)
		o.canvas.SetContentHeight(height)
	}
}

func (o *captureOverlay) cursorName() string {
	if o.mode == CaptureWindow && o.tool == annotate.ToolSelect && len(o.frames) > 1 {
		return "pointer"
	}
	return cursorForTool(o.tool)
}

func mapWindowFrames(frames []grab.WindowFrame, img image.Point, monitor image.Rectangle) []image.Rectangle {
	out := make([]image.Rectangle, 0, len(frames))
	for _, frame := range frames {
		r := grab.MapToImage(frame.Bounds, img, monitor)
		if r.Dx() >= 32 && r.Dy() >= 32 {
			out = append(out, r)
		}
	}
	return out
}

func monitorRect(win *gtk.ApplicationWindow) image.Rectangle {
	_, _, monitor := displayBounds(win)
	if monitor == nil {
		return image.Rectangle{}
	}
	geo := monitor.Geometry()
	if geo == nil {
		return image.Rectangle{}
	}
	return image.Rect(geo.X(), geo.Y(), geo.X()+geo.Width(), geo.Y()+geo.Height())
}

func drawWindowHover(cr *cairo.Context, r image.Rectangle) {
	if r.Empty() {
		return
	}
	cr.SetSourceRGBA(0.21, 0.52, 0.96, 0.16)
	cr.Rectangle(float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()))
	cr.Fill()
	cr.SetSourceRGB(0.45, 0.72, 1)
	cr.SetLineWidth(2 / maxFloat(crScale(cr), 0.5))
	cr.Rectangle(float64(r.Min.X)+0.5, float64(r.Min.Y)+0.5, float64(r.Dx()-1), float64(r.Dy()-1))
	cr.Stroke()
}

func displayBounds(win *gtk.ApplicationWindow) (int, int, *gdk.Monitor) {
	display := win.Root.Display()
	if display == nil {
		display = gdk.DisplayGetDefault()
	}
	if display == nil {
		return 0, 0, nil
	}

	var monitor *gdk.Monitor
	if surface := win.Root.Surface(); surface != nil {
		monitor = display.MonitorAtSurface(surface)
	}
	if monitor == nil {
		monitor = firstMonitor(display)
	}
	if monitor == nil {
		return 0, 0, nil
	}
	geo := monitor.Geometry()
	if geo == nil || geo.Width() < 1 || geo.Height() < 1 {
		return 0, 0, monitor
	}
	return geo.Width(), geo.Height(), monitor
}

func firstMonitor(display *gdk.Display) *gdk.Monitor {
	model := display.Monitors()
	if model == nil || model.NItems() == 0 {
		return nil
	}
	item := model.Item(0)
	if item == nil {
		return nil
	}
	casted := item.WalkCast(func(obj glib.Objector) bool {
		_, ok := obj.(*gdk.Monitor)
		return ok
	})
	monitor, _ := casted.(*gdk.Monitor)
	return monitor
}
