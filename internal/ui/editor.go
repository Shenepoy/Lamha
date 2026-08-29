package ui

import (
	"fmt"
	"math"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/annotate"
	"github.com/lamha-app/lamha/internal/brand"
	"github.com/lamha-app/lamha/internal/capture"
	"github.com/lamha-app/lamha/internal/i18n"
	"github.com/lamha-app/lamha/internal/keys"
)

var editorColors = []struct {
	name  string
	color annotate.Color
}{
	{"Red", annotate.RGB(0xe11d48)},
	{"Orange", annotate.RGB(0xea580c)},
	{"Yellow", annotate.RGB(0xfacc15)},
	{"Green", annotate.RGB(0x16a34a)},
	{"Blue", annotate.RGB(0x2563eb)},
	{"White", annotate.RGB(0xffffff)},
	{"Black", annotate.RGB(0x111827)},
}

func colorName(name string) string {
	return i18n.T(name)
}

type editor struct {
	parent        *Window
	window        *gtk.Window
	canvas        *gtk.DrawingArea
	status        *gtk.Label
	copyOnSave    *gtk.CheckButton
	undoButton    *gtk.Button
	redoButton    *gtk.Button
	sizeScale     *gtk.Scale
	tools         *toolSwitcher
	doc           *annotate.Document
	path          string
	surface       *cairo.Surface
	tool          annotate.Tool
	color         annotate.Color
	width         float64
	draft         *annotate.Stroke
	view          annotate.View
	selected      int
	moving        bool
	moveLast      annotate.Point
	cursorX       float64
	cursorY       float64
	cursorIn      bool
	magnifierSize float64
	host          *gtk.Overlay
	text          textInput
	redraw        redrawPump
	lens          *magnifierLens
}

func (w *Window) openEditor(path string) {
	if w.editor != nil {
		w.editor.window.Destroy()
		w.editor = nil
	}

	doc, err := annotate.Open(path)
	if err != nil {
		w.status.SetText(i18n.Tf("Could not open capture for annotation: %v", err))
		return
	}

	ed := &editor{
		parent:        w,
		doc:           doc,
		path:          path,
		tool:          annotate.ToolPen,
		color:         editorColors[0].color,
		width:         8,
		selected:      -1,
		magnifierSize: defaultMagnifier,
	}
	ed.build()
	ed.refreshSurface()
	ed.window.Present()
	w.editor = ed
}

func (e *editor) build() {
	e.window = gtk.NewWindow()
	if app := e.parent.window.Application(); app != nil {
		e.window.SetApplication(app)
	}
	e.window.SetTitle(i18n.T("Annotate capture"))
	e.window.SetIconName(brand.Name)
	applyDirection(&e.window.Widget)
	e.window.SetDefaultSize(1100, 760)
	e.window.SetTransientFor(&e.parent.window.Window)
	e.window.SetModal(true)
	e.window.SetDestroyWithParent(true)
	e.window.ConnectCloseRequest(func() bool {
		e.text.cancel()
		if e.lens != nil {
			e.lens.hide()
		}
		e.parent.editor = nil
		return false
	})

	keysCtl := gtk.NewEventControllerKey()
	keysCtl.SetPropagationPhase(gtk.PhaseCapture)
	keysCtl.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		if e.text.active() {
			if keyval == gdk.KEY_Escape {
				e.text.cancel()
				e.refreshSurface()
				e.status.SetText(i18n.T("Click a mark to move or restyle it. Double-click text to edit. Scroll resizes the lens. Shortcuts are editable in the main window."))
				e.canvas.QueueDraw()
				return true
			}
			return false
		}
		return handleAnnotateKey(keyval, state, e.keyActions())
	})
	e.window.AddController(keysCtl)

	header := gtk.NewHeaderBar()
	e.window.SetTitlebar(header)

	ensureToolbarCSS()

	e.undoButton = newIconButton("edit-undo-symbolic", withKey(i18n.T("Undo"), keys.Undo))
	e.undoButton.SetSensitive(false)
	e.undoButton.ConnectClicked(e.undo)
	header.PackStart(e.undoButton)

	e.redoButton = newIconButton("edit-redo-symbolic", withKey(i18n.T("Redo"), keys.Redo))
	e.redoButton.SetSensitive(false)
	e.redoButton.ConnectClicked(e.redo)
	header.PackStart(e.redoButton)

	save := newIconButton("document-save-symbolic", withKey(i18n.T("Save"), keys.Save))
	save.SetCSSClasses([]string{"suggested-action"})
	save.ConnectClicked(e.save)
	header.PackEnd(save)

	root := gtk.NewBox(gtk.OrientationVertical, 10)
	root.SetMarginTop(12)
	root.SetMarginBottom(12)
	root.SetMarginStart(12)
	root.SetMarginEnd(12)

	tools := gtk.NewBox(gtk.OrientationHorizontal, 2)
	e.tools = appendToolToggles(tools, editorTools(), e.tool, e.setTool)
	root.Append(tools)

	options := gtk.NewBox(gtk.OrientationHorizontal, 8)
	for i, swatch := range editorColors {
		i := i
		options.Append(newColorSwatch(withKey(colorName(swatch.name), colorID(i)), swatch.color, func(annotate.Color) {
			e.setColorIndex(i)
		}))
	}

	options.Append(gtk.NewLabel(i18n.T("Size")))
	e.sizeScale = gtk.NewScaleWithRange(gtk.OrientationHorizontal, 2, 28, 1)
	e.sizeScale.SetDrawValue(true)
	e.sizeScale.SetValue(e.width)
	e.sizeScale.SetSizeRequest(140, -1)
	e.sizeScale.ConnectValueChanged(func() {
		e.setWidth(e.sizeScale.Value())
	})
	options.Append(e.sizeScale)

	e.copyOnSave = gtk.NewCheckButtonWithLabel(i18n.T("Also copy to clipboard when saving"))
	e.copyOnSave.SetActive(true)
	options.Append(e.copyOnSave)
	root.Append(options)

	e.status = gtk.NewLabel(i18n.T("Click a mark to move or restyle it. Double-click text to edit. Scroll resizes the lens. Shortcuts are editable in the main window."))
	alignStart(e.status)
	e.status.SetCSSClasses([]string{"dim-label"})
	root.Append(e.status)

	e.canvas = gtk.NewDrawingArea()
	e.canvas.SetHExpand(true)
	e.canvas.SetVExpand(true)
	e.canvas.SetCursorFromName(cursorForTool(e.tool))
	e.canvas.SetDrawFunc(e.draw)
	e.bindGestures()
	bindCursorTracking(e.canvas, func(x, y float64) {
		e.cursorX, e.cursorY = x, y
		e.cursorIn = true
		e.updateLens()
	}, func() {
		e.cursorIn = false
		e.updateLens()
	}, func(dy float64) {
		e.magnifierSize = clampMagnifier(e.magnifierSize - dy*18)
		if e.lens != nil {
			e.lens.setSize(e.magnifierSize)
		}
		e.updateLens()
	})

	e.host = gtk.NewOverlay()
	e.host.SetDirection(gtk.TextDirLTR)
	e.host.SetChild(e.canvas)
	e.lens = newMagnifierLens()
	e.lens.setSize(e.magnifierSize)
	e.lens.attach(e.host)

	frame := gtk.NewFrame("")
	frame.SetHExpand(true)
	frame.SetVExpand(true)
	frame.SetChild(e.host)
	root.Append(frame)

	e.window.SetChild(root)
}

func (e *editor) bindGestures() {
	click := gtk.NewGestureClick()
	click.SetButton(gdk.BUTTON_PRIMARY)
	click.ConnectPressed(func(nPress int, x, y float64) {
		if nPress >= 2 && e.beginTextEdit(e.view.ToImage(x, y)) {
			click.SetState(gtk.EventSequenceClaimed)
		}
	})
	click.ConnectReleased(func(nPress int, x, y float64) {
		if nPress < 1 || e.text.active() {
			return
		}
		point := e.view.ToImage(x, y)
		if e.tool == annotate.ToolSelect || e.tool == annotate.ToolMove {
			e.selected = e.doc.Hit(point)
			e.refreshActions()
			e.canvas.QueueDraw()
			return
		}
		if e.tool == annotate.ToolText {
			e.commitTyping()
			e.text.begin(e.host, e.view, point, e.color, e.width, func() {
				e.commitTyping()
				e.refreshSurface()
				e.canvas.QueueDraw()
				e.status.SetText(i18n.T("Click a mark to move or restyle it. Double-click text to edit. Scroll resizes the lens. Shortcuts are editable in the main window."))
			})
			e.status.SetText(i18n.T("Type, then Enter to place. Escape cancels."))
			e.updateLens()
			e.canvas.QueueDraw()
			return
		}
		if e.tool != annotate.ToolStep {
			return
		}
		e.selected = -1
		e.doc.Add(annotate.Stroke{
			Tool:   annotate.ToolStep,
			Color:  e.color,
			Width:  e.width,
			X1:     point.X,
			Y1:     point.Y,
			Points: []annotate.Point{point},
			Step:   e.doc.NextStep(),
		})
		e.refreshSurface()
		e.canvas.QueueDraw()
	})
	e.canvas.AddController(click)

	drag := gtk.NewGestureDrag()
	drag.SetButton(gdk.BUTTON_PRIMARY)
	drag.ConnectDragBegin(func(startX, startY float64) {
		if e.text.active() {
			return
		}
		start := e.view.ToImage(startX, startY)
		if e.tool == annotate.ToolSelect || e.tool == annotate.ToolMove {
			if hit := e.doc.Hit(start); hit >= 0 {
				e.selected = hit
				e.moving = true
				e.moveLast = start
				e.draft = nil
				e.beginLiveMove()
				e.canvas.SetCursorFromName("grabbing")
				e.canvas.QueueDraw()
				return
			}
			if e.tool == annotate.ToolMove && e.selected >= 0 {
				e.moving = true
				e.moveLast = start
				e.beginLiveMove()
				e.canvas.SetCursorFromName("grabbing")
			} else {
				e.selected = -1
			}
			e.draft = nil
			e.canvas.QueueDraw()
			return
		}
		if toolPlacesOnClick(e.tool) {
			return
		}
		e.selected = -1
		e.draft = &annotate.Stroke{
			Tool:   e.tool,
			Color:  e.color,
			Width:  e.width,
			X1:     start.X,
			Y1:     start.Y,
			X2:     start.X,
			Y2:     start.Y,
			Points: []annotate.Point{start},
		}
		e.canvas.QueueDraw()
	})
	drag.ConnectDragUpdate(func(offsetX, offsetY float64) {
		startX, startY, ok := drag.StartPoint()
		if !ok {
			return
		}
		point := e.view.ToImage(startX+offsetX, startY+offsetY)
		if e.moving && e.selected >= 0 {
			e.doc.Move(e.selected, point.X-e.moveLast.X, point.Y-e.moveLast.Y)
			e.moveLast = point
			e.queueFrame()
			return
		}
		if e.draft == nil {
			return
		}
		e.draft.X2 = point.X
		e.draft.Y2 = point.Y
		if e.draft.Tool == annotate.ToolPen || e.draft.Tool == annotate.ToolMagicErase {
			e.draft.Points = append(e.draft.Points, point)
		}
		e.queueFrame()
	})
	drag.ConnectDragEnd(func(offsetX, offsetY float64) {
		if e.moving {
			e.moving = false
			e.refreshSurface()
			e.canvas.SetCursorFromName(cursorForTool(e.tool))
			e.canvas.QueueDraw()
			e.status.SetText(i18n.Tf("Moved the selected mark. Arrow keys nudge. %s.", withKey(i18n.T("Duplicate"), keys.Duplicate)))
			return
		}
		if e.draft == nil {
			return
		}
		stroke := *e.draft
		e.draft = nil
		if !strokeReady(stroke) {
			return
		}
		e.doc.Add(stroke)
		e.refreshSurface()
		e.canvas.QueueDraw()
	})
	e.canvas.AddController(drag)
}

func (e *editor) draw(_ *gtk.DrawingArea, cr *cairo.Context, width, height int) {
	size := e.doc.Size()
	e.view = annotate.Fit(size.X, size.Y, width, height)

	cr.SetSourceRGB(0.11, 0.12, 0.14)
	cr.Paint()
	if e.surface == nil {
		return
	}

	cr.Save()
	cr.Translate(e.view.OffsetX, e.view.OffsetY)
	cr.Scale(e.view.Scale, e.view.Scale)
	paintSurface(cr, e.surface, cairo.FilterGood)
	if stroke, ok := e.doc.Stroke(e.selected); ok {
		if e.moving {
			drawDraft(cr, stroke)
		}
		drawSelectedStroke(cr, stroke)
	}
	if e.text.active() && e.text.entry == nil && e.text.draft.text != "" {
		drawDraft(cr, e.text.draft.stroke())
	}
	if e.draft != nil {
		drawDraft(cr, *e.draft)
	}
	cr.Restore()
}

func drawDraft(cr *cairo.Context, stroke annotate.Stroke) {
	r := float64(stroke.Color.R) / 255
	g := float64(stroke.Color.G) / 255
	b := float64(stroke.Color.B) / 255
	width := math.Max(stroke.Width, 1)

	switch stroke.Tool {
	case annotate.ToolPen, annotate.ToolMagicErase:
		if stroke.Tool == annotate.ToolMagicErase {
			cr.SetSourceRGBA(0.85, 0.2, 0.35, 0.4)
		} else {
			cr.SetSourceRGB(r, g, b)
		}
		drawSmoothPath(cr, stroke.Points, width)
	case annotate.ToolArrow:
		cr.SetSourceRGB(r, g, b)
		cr.SetLineWidth(width)
		cr.SetLineCap(cairo.LineCapRound)
		cr.MoveTo(stroke.X1, stroke.Y1)
		cr.LineTo(stroke.X2, stroke.Y2)
		cr.Stroke()
		drawArrowHead(cr, stroke.X1, stroke.Y1, stroke.X2, stroke.Y2, width)
	case annotate.ToolEllipse:
		x, y, w, h := draftRect(stroke)
		cr.SetSourceRGB(r, g, b)
		cr.SetLineWidth(width)
		drawEllipsePath(cr, x, y, w, h)
		cr.Stroke()
	case annotate.ToolText:
		annotate.PaintText(cr, stroke.X1, stroke.Y1, stroke.Text, r, g, b, width)
	case annotate.ToolBox, annotate.ToolBlur, annotate.ToolAreaErase:
		x, y, w, h := draftRect(stroke)
		cr.SetSourceRGB(r, g, b)
		cr.SetLineWidth(width)
		if stroke.Tool != annotate.ToolBox {
			cr.SetDash([]float64{8, 6}, 0)
		}
		if stroke.Tool == annotate.ToolAreaErase {
			cr.SetSourceRGBA(0.85, 0.2, 0.35, 0.28)
			cr.Rectangle(x, y, w, h)
			cr.Fill()
			cr.SetSourceRGB(0.85, 0.2, 0.35)
		}
		cr.Rectangle(x, y, w, h)
		cr.Stroke()
		cr.SetDash(nil, 0)
	case annotate.ToolHighlight:
		x, y, w, h := draftRect(stroke)
		cr.SetSourceRGBA(r, g, b, 0.38)
		cr.Rectangle(x, y, w, h)
		cr.Fill()
	}
}

func strokeReady(stroke annotate.Stroke) bool {
	switch stroke.Tool {
	case annotate.ToolBox, annotate.ToolHighlight, annotate.ToolBlur, annotate.ToolAreaErase, annotate.ToolArrow, annotate.ToolEllipse:
		return math.Hypot(stroke.X2-stroke.X1, stroke.Y2-stroke.Y1) >= 2
	case annotate.ToolText:
		return stroke.Text != ""
	default:
		return len(stroke.Points) > 0
	}
}

func draftRect(stroke annotate.Stroke) (x, y, w, h float64) {
	x1, x2 := stroke.X1, stroke.X2
	y1, y2 := stroke.Y1, stroke.Y2
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	return x1, y1, x2 - x1, y2 - y1
}

func drawEllipsePath(cr *cairo.Context, x, y, w, h float64) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	cr.Save()
	cr.Translate(x+w/2, y+h/2)
	cr.Scale(w/2, h/2)
	cr.Arc(0, 0, 1, 0, 2*math.Pi)
	cr.Restore()
}

func drawArrowHead(cr *cairo.Context, x1, y1, x2, y2, width float64) {
	angle := math.Atan2(y2-y1, x2-x1)
	head := math.Max(width*4.2, 16)
	cr.MoveTo(x2, y2)
	cr.LineTo(x2+math.Cos(angle+math.Pi*0.82)*head, y2+math.Sin(angle+math.Pi*0.82)*head)
	cr.LineTo(x2+math.Cos(angle-math.Pi*0.82)*head, y2+math.Sin(angle-math.Pi*0.82)*head)
	cr.ClosePath()
	cr.Fill()
}

func (e *editor) queueFrame() {
	e.redraw.request(func() {
		if e.canvas != nil {
			e.canvas.QueueDraw()
		}
	})
}

func (e *editor) beginLiveMove() {
	if e.selected < 0 {
		return
	}
	e.surface = imageSurface(e.doc.RenderExcept(e.selected))
	e.updateLens()
}

func (e *editor) skipStroke() int {
	if e.moving && e.selected >= 0 {
		return e.selected
	}
	if e.text.active() && e.text.replace >= 0 {
		return e.text.replace
	}
	return -1
}

func (e *editor) refreshSurface() {
	if skip := e.skipStroke(); skip >= 0 {
		e.surface = imageSurface(e.doc.RenderExcept(skip))
	} else {
		e.surface = imageSurface(e.doc.Raster())
	}
	e.refreshActions()
	e.updateLens()
}

func (e *editor) updateLens() {
	if e.lens == nil || e.canvas == nil || e.doc == nil {
		return
	}
	if !e.cursorIn || e.moving || e.draft != nil || e.text.active() {
		e.lens.hide()
		return
	}
	size := e.doc.Size()
	e.view = annotate.Fit(size.X, size.Y, e.canvas.AllocatedWidth(), e.canvas.AllocatedHeight())
	e.lens.follow(e.view, e.doc.Raster(), e.cursorX, e.cursorY, e.canvas.AllocatedWidth(), e.canvas.AllocatedHeight())
}

func (e *editor) beginTextEdit(point annotate.Point) bool {
	if e.tool != annotate.ToolMove && e.tool != annotate.ToolSelect {
		return false
	}
	hit := e.doc.Hit(point)
	stroke, ok := e.doc.Stroke(hit)
	if !ok || stroke.Tool != annotate.ToolText {
		return false
	}
	if e.moving {
		e.moving = false
	}
	e.commitTyping()
	e.selected = hit
	e.color = stroke.Color
	e.width = stroke.Width
	if e.sizeScale != nil && e.sizeScale.Value() != e.width {
		e.sizeScale.SetValue(e.width)
	}
	e.text.beginReplace(e.host, e.view, stroke, hit, func() {
		e.selected = applyTextCommit(e.doc, &e.text)
		e.refreshSurface()
		e.canvas.QueueDraw()
		e.status.SetText(i18n.T("Click a mark to move or restyle it. Double-click text to edit. Scroll resizes the lens. Shortcuts are editable in the main window."))
	})
	e.refreshSurface()
	e.canvas.QueueDraw()
	e.status.SetText(i18n.T("Edit text, then Enter to save. Escape cancels."))
	return true
}

func (e *editor) refreshActions() {
	if e.undoButton != nil {
		e.undoButton.SetSensitive(e.doc.CanUndo())
	}
	if e.redoButton != nil {
		e.redoButton.SetSensitive(e.doc.CanRedo())
	}
}

func (e *editor) setTool(tool annotate.Tool) {
	if e.tool != tool {
		e.commitTyping()
	}
	e.tool = tool
	e.draft = nil
	if e.tools != nil {
		e.tools.activate(tool)
	}
	if e.canvas != nil {
		e.canvas.SetCursorFromName(cursorForTool(tool))
		e.canvas.QueueDraw()
	}
}

func (e *editor) setColorIndex(index int) {
	if index < 0 || index >= len(editorColors) {
		return
	}
	e.color = editorColors[index].color
	if e.text.active() {
		e.text.draft.color = e.color
		e.canvas.QueueDraw()
	}
	if stroke, ok := e.doc.Stroke(e.selected); ok {
		stroke.Color = e.color
		e.doc.Replace(e.selected, stroke)
		e.refreshSurface()
		e.canvas.QueueDraw()
	}
}

func (e *editor) setWidth(width float64) {
	if width < 2 {
		width = 2
	}
	if width > 28 {
		width = 28
	}
	e.width = width
	if e.sizeScale != nil && e.sizeScale.Value() != width {
		e.sizeScale.SetValue(width)
	}
	if e.text.active() {
		e.text.draft.width = width
		e.canvas.QueueDraw()
	}
	if stroke, ok := e.doc.Stroke(e.selected); ok {
		stroke.Width = width
		e.doc.Replace(e.selected, stroke)
		e.refreshSurface()
		e.canvas.QueueDraw()
	}
}

func (e *editor) keyActions() keyActions {
	return keyActions{
		setTool:    e.setTool,
		setColor:   e.setColorIndex,
		nudgeWidth: func(delta float64) { e.setWidth(e.width + delta) },
		nudge:      e.nudgeSelected,
		undo:       e.undo,
		redo:       e.redo,
		delete:     e.deleteSelected,
		duplicate:  e.duplicateSelected,
		save:       e.save,
		escape: func() bool {
			if e.text.active() {
				e.text.cancel()
				e.refreshSurface()
				e.status.SetText(i18n.T("Click a mark to move or restyle it. Double-click text to edit. Scroll resizes the lens. Shortcuts are editable in the main window."))
				e.canvas.QueueDraw()
				return true
			}
			if e.selected >= 0 {
				e.selected = -1
				e.canvas.QueueDraw()
				return true
			}
			return false
		},
	}
}

func (e *editor) nudgeSelected(dx, dy float64) bool {
	if e.selected < 0 {
		return false
	}
	if !e.doc.Move(e.selected, dx, dy) {
		return false
	}
	e.refreshSurface()
	e.canvas.QueueDraw()
	return true
}

func (e *editor) duplicateSelected() {
	next := e.doc.Duplicate(e.selected)
	if next < 0 {
		return
	}
	e.selected = next
	e.refreshSurface()
	e.canvas.QueueDraw()
	e.status.SetText(i18n.T("Duplicated the selected mark."))
}

func (e *editor) commitTyping() {
	if !e.text.active() {
		return
	}
	e.selected = applyTextCommit(e.doc, &e.text)
	e.refreshSurface()
}

func (e *editor) deleteSelected() {
	if !e.doc.Remove(e.selected) {
		return
	}
	e.selected = -1
	e.refreshSurface()
	e.canvas.QueueDraw()
	e.status.SetText(i18n.T("Removed the selected mark."))
}

func (e *editor) undo() {
	if !e.doc.Undo() {
		return
	}
	e.selected = -1
	e.refreshSurface()
	e.canvas.QueueDraw()
	e.status.SetText(i18n.T("Undid the last mark."))
}

func (e *editor) redo() {
	if !e.doc.Redo() {
		return
	}
	e.selected = -1
	e.refreshSurface()
	e.canvas.QueueDraw()
	e.status.SetText(i18n.T("Redid the last mark."))
}

func (e *editor) save() {
	e.commitTyping()
	rendered := e.doc.Render()
	if err := capture.WritePNG(e.path, rendered); err != nil {
		e.status.SetText(i18n.Tf("Could not save: %v", err))
		return
	}
	if e.copyOnSave.Active() {
		if err := copyImageFile(e.path); err != nil {
			e.status.SetText(i18n.Tf("Saved, but clipboard copy failed: %v", err))
			e.parent.afterAnnotation(e.path, false)
			return
		}
		e.status.SetText(i18n.T("Saved and copied to the clipboard."))
		e.parent.afterAnnotation(e.path, true)
		return
	}
	e.status.SetText(i18n.T("Saved annotated capture."))
	e.parent.afterAnnotation(e.path, false)
}

func copyImageFile(path string) error {
	texture, err := gdk.NewTextureFromFilename(path)
	if err != nil {
		return err
	}
	display := gdk.DisplayGetDefault()
	if display == nil {
		return fmt.Errorf("no display is available for the clipboard")
	}
	display.Clipboard().SetTexture(texture)
	return nil
}
