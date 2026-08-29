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
	selStart      annotate.Point
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
	frames        []image.Rectangle
	hovered       image.Rectangle
	windowLocked  bool
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
	}
	switch mode {
	case CaptureScreen:
		size := doc.Size()
		ov.selection = image.Rect(0, 0, size.X, size.Y)
	case CaptureWindow:
		if mon.Empty() {
			mon = monitorRect(w.window)
		}
		ov.frames = mapWindowFrames(frames, doc.Size(), mon)
	}
	ov.build()
	ov.refreshSurface()
	w.overlay = ov

	width, height, monitor := displayBounds(w.window)
	if !mon.Empty() {
		width, height = mon.Dx(), mon.Dy()
	}
	ov.coverMonitor(width, height)
	ov.window.Present()
	if monitor != nil {
		glib.TimeoutAdd(80, func() bool {
			if ov.window == nil {
				return false
			}
			if !ov.window.IsFullscreen() {
				ov.window.FullscreenOnMonitor(monitor)
			}
			if ov.canvas != nil {
				ov.canvas.QueueDraw()
			}
			return false
		})
	}
	w.window.SetVisible(false)
}

func (o *captureOverlay) build() {
	o.window = gtk.NewWindow()
	if app := o.parent.window.Application(); app != nil {
		o.window.SetApplication(app)
	}
	o.window.SetTitle(i18n.T("Lamha capture"))
	o.window.SetIconName(brand.Name)
	applyDirection(&o.window.Widget)
	o.window.SetDecorated(false)
	o.window.SetDeletable(false)
	o.window.SetModal(true)
	o.window.ConnectMap(func() {
		if o.canvas != nil {
			o.canvas.GrabFocus()
			o.canvas.QueueDraw()
		}
	})
	o.window.ConnectCloseRequest(func() bool {
		o.close(true)
		return true
	})

	keysCtl := gtk.NewEventControllerKey()
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
	o.canvas.SetHExpand(true)
	o.canvas.SetVExpand(true)
	o.canvas.SetFocusable(true)
	o.canvas.SetCursorFromName(o.cursorName())
	o.canvas.SetDrawFunc(o.draw)
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
	}, func() {
		o.cursorIn = false
		if !o.hovered.Empty() {
			o.hovered = image.Rectangle{}
			o.canvas.QueueDraw()
		}
		o.updateLens()
	}, func(dy float64) {
		o.magnifierSize = clampMagnifier(o.magnifierSize - dy*18)
		if o.lens != nil {
			o.lens.setSize(o.magnifierSize)
		}
		o.updateLens()
	})

	o.host = gtk.NewOverlay()
	o.host.SetDirection(gtk.TextDirLTR)
	o.host.SetChild(o.canvas)
	o.lens = newMagnifierLens()
	o.lens.setSize(o.magnifierSize)
	o.lens.attach(o.host)
	chrome := o.buildChrome()
	if i18n.RTL() {
		chrome.SetDirection(gtk.TextDirRTL)
	}
	o.host.AddOverlay(chrome)

	o.window.SetChild(o.host)
}

func (o *captureOverlay) buildChrome() *gtk.Box {
	ensureToolbarCSS()

	chrome := gtk.NewBox(gtk.OrientationHorizontal, 6)
	chrome.SetHAlign(gtk.AlignCenter)
	chrome.SetVAlign(gtk.AlignStart)
	chrome.SetHExpand(false)
	chrome.SetCSSClasses([]string{"lamha-toolbar"})

	tools := gtk.NewBox(gtk.OrientationHorizontal, 2)
	tools.SetVAlign(gtk.AlignCenter)
	o.tools = appendToolToggles(tools, captureTools(), o.tool, iconInkOnDark, o.setTool)
	chrome.Append(tools)

	chrome.Append(gtk.NewSeparator(gtk.OrientationVertical))

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
	o.status.SetCSSClasses([]string{"dim-label"})
	if i18n.RTL() {
		o.status.SetDirection(gtk.TextDirRTL)
	}

	shell := gtk.NewBox(gtk.OrientationVertical, 6)
	shell.SetHAlign(gtk.AlignCenter)
	shell.SetVAlign(gtk.AlignStart)
	shell.Append(chrome)
	shell.Append(o.status)
	return shell
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
		return i18n.T("Drag to select. Scroll zooms the lens. Shortcuts are editable in the main window.")
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
			o.selected = o.doc.Hit(point)
			if stroke, ok := o.doc.Stroke(o.selected); ok {
				o.adoptStroke(stroke)
			}
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
			o.selected = -1
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
		if o.selecting {
			o.selection = annotate.RectFromPoints(o.selStart.X, o.selStart.Y, point.X, point.Y)
			o.queueFrame()
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
			o.refreshSurface()
			o.canvas.SetCursorFromName(o.cursorName())
			o.canvas.QueueDraw()
			o.status.SetText(i18n.Tf("Moved the selected mark. Arrow keys nudge. %s.", withKey(i18n.T("Duplicate"), keys.Duplicate)))
			return
		}
		if o.selecting {
			o.selecting = false
			if o.selection.Dx() < 2 || o.selection.Dy() < 2 {
				o.selection = image.Rectangle{}
			} else if o.mode == CaptureWindow && !o.windowLocked {
				o.lockWindow(o.selection)
				return
			}
			o.status.SetText(o.hint())
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
	if o.draft != nil || o.selecting {
		o.draft = nil
		o.selecting = false
		if o.mode == CaptureArea {
			o.selection = image.Rectangle{}
		}
		o.status.SetText(o.hint())
		o.canvas.QueueDraw()
		return
	}
	if !o.selection.Empty() {
		o.selection = image.Rectangle{}
		o.status.SetText(o.hint())
		o.canvas.QueueDraw()
		return
	}
	o.close(true)
}

func (o *captureOverlay) draw(_ *gtk.DrawingArea, cr *cairo.Context, width, height int) {
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
	drawSelectionMask(cr, size, o.selection)
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

func drawSelectionMask(cr *cairo.Context, size image.Point, selection image.Rectangle) {
	if selection.Empty() {
		cr.SetSourceRGBA(0, 0, 0, 0.35)
		cr.Rectangle(0, 0, float64(size.X), float64(size.Y))
		cr.Fill()
		return
	}
	sel := selection.Intersect(image.Rect(0, 0, size.X, size.Y))
	if sel.Empty() {
		return
	}
	cr.SetSourceRGBA(0, 0, 0, 0.5)
	cr.Rectangle(0, 0, float64(size.X), float64(sel.Min.Y))
	cr.Rectangle(0, float64(sel.Min.Y), float64(sel.Min.X), float64(sel.Dy()))
	cr.Rectangle(float64(sel.Max.X), float64(sel.Min.Y), float64(size.X-sel.Max.X), float64(sel.Dy()))
	cr.Rectangle(0, float64(sel.Max.Y), float64(size.X), float64(size.Y-sel.Max.Y))
	cr.Fill()

	cr.SetSourceRGB(1, 1, 1)
	cr.SetLineWidth(2 / maxFloat(crScale(cr), 0.5))
	cr.Rectangle(float64(sel.Min.X)+0.5, float64(sel.Min.Y)+0.5, float64(sel.Dx()-1), float64(sel.Dy()-1))
	cr.Stroke()
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
	if !o.cursorIn || o.moving || o.selecting || o.draft != nil || o.text.active() {
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
	if o.tools != nil {
		o.tools.activate(tool)
	}
	if o.canvas != nil {
		o.canvas.SetCursorFromName(o.cursorName())
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
	o.canvas.QueueDraw()
	return true
}

func (o *captureOverlay) duplicateSelected() {
	next := o.doc.Duplicate(o.selected)
	if next < 0 {
		return
	}
	o.selected = next
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
	o.refreshSurface()
	o.canvas.QueueDraw()
	o.status.SetText(i18n.T("Undid the last mark."))
}

func (o *captureOverlay) redo() {
	if !o.doc.Redo() {
		return
	}
	o.selected = -1
	o.refreshSurface()
	o.canvas.QueueDraw()
	o.status.SetText(i18n.T("Redid the last mark."))
}

func (o *captureOverlay) confirm() {
	o.commitTyping()
	rendered := o.doc.Render()
	if !o.selection.Empty() {
		rendered = annotate.Crop(rendered, o.selection)
	}
	saved, err := o.parent.store.SaveImage(rendered)
	if err != nil {
		o.status.SetText(i18n.Tf("Could not save: %s", err.Error()))
		return
	}
	message := i18n.Tf("Saved %s", saved.Path)
	if o.copyOnSave.Active() {
		if err := copyImageFile(saved.Path); err != nil {
			message = i18n.Tf("Saved, but clipboard copy failed: %s", err.Error())
		} else {
			message = i18n.T("Saved and copied to the clipboard.")
		}
	}
	o.dismiss(func() {
		o.parent.showCapture(saved, message)
		o.parent.refreshHistory(saved.Path)
		if o.parent.restoreAfterCapture {
			o.parent.Present()
		} else {
			o.parent.Notify("Lamha", message)
		}
	})
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
	win.SetVisible(false)
	win.Destroy()
	if after != nil {
		after()
	}
}

func (o *captureOverlay) lockWindow(frame image.Rectangle) {
	if o.windowLocked || o.staging == "" || frame.Empty() {
		o.selection = frame
		o.hovered = frame
		o.selecting = false
		o.draft = nil
		o.status.SetText(o.hint())
		o.canvas.QueueDraw()
		return
	}
	if err := grab.CropPNG(o.staging, frame); err != nil {
		log.Printf("window crop: %v", err)
		o.selection = frame
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
		o.selection = frame
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
	o.selection = image.Rect(0, 0, size.X, size.Y)
	o.refreshSurface()
	o.canvas.SetCursorFromName(o.cursorName())
	o.status.SetText(o.hint())
	o.canvas.QueueDraw()
}

func (o *captureOverlay) coverMonitor(width, height int) {
	if width < 1 || height < 1 {
		size := o.doc.Size()
		width, height = size.X, size.Y
	}
	if width < 1 {
		width, height = 1920, 1080
	}
	o.window.SetDefaultSize(width, height)
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
