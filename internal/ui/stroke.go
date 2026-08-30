package ui

import (
	"fmt"
	"math"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/i18n"
	"github.com/lamha-app/lamha/internal/keys"
)

const (
	minStrokeWidth = 1
	maxStrokeWidth = 48

	compactPreviewLength  = 36
	compactPreviewBreadth = 36
	verticalPreviewSize   = 32
	editorPreviewLength   = 64
	editorPreviewBreadth  = 18
	strokePreviewPad      = 3.0
)

type strokeControl struct {
	box        *gtk.Box
	stepper    *gtk.Box
	scale      *gtk.Scale
	preview    *gtk.DrawingArea
	adjustment *gtk.Adjustment
	value      *gtk.Label
	updating   bool
	compact    bool
	dark       bool
	vertical   bool
}

func strokeWidthTip() string {
	return withKey(i18n.T("Stroke width"), keys.WidthDown) + " / " + accelLabel(keys.Current().Accel(keys.WidthUp))
}

func clampStroke(width float64) float64 {
	width = math.Round(width)
	if width < minStrokeWidth {
		return minStrokeWidth
	}
	if width > maxStrokeWidth {
		return maxStrokeWidth
	}
	return width
}

func newStrokeControl(value float64, compact, dark bool, tooltip string, onChange func(float64)) *strokeControl {
	value = clampStroke(value)
	adj := gtk.NewAdjustment(value, minStrokeWidth, maxStrokeWidth, 1, 4, 0)

	box := gtk.NewBox(gtk.OrientationHorizontal, 6)
	box.SetVAlign(gtk.AlignCenter)
	box.SetHAlign(gtk.AlignCenter)
	if !compact {
		label := gtk.NewLabel(i18n.T("Stroke"))
		label.SetCSSClasses([]string{"dim-label", "dimmed"})
		box.Append(label)
	}

	ctrl := &strokeControl{box: box, adjustment: adj, compact: compact, dark: dark}

	preview := gtk.NewDrawingArea()
	preview.SetHAlign(gtk.AlignCenter)
	preview.SetVAlign(gtk.AlignCenter)
	preview.SetCSSClasses([]string{"lamha-stroke-preview"})
	preview.SetTooltipText(tooltip)
	preview.SetDrawFunc(func(_ *gtk.DrawingArea, cr *cairo.Context, width, height int) {
		drawStrokePreview(cr, width, height, adj.Value(), ctrl.dark, ctrl.vertical)
	})
	unfocusable(&preview.Widget)
	box.Append(preview)

	scale := gtk.NewScale(gtk.OrientationHorizontal, adj)
	scale.SetDrawValue(false)
	scale.SetHAlign(gtk.AlignCenter)
	scale.SetVAlign(gtk.AlignCenter)
	scale.SetTooltipText(tooltip)
	unfocusable(&scale.Widget)
	box.Append(scale)

	stepper := gtk.NewBox(gtk.OrientationHorizontal, 0)
	stepper.SetHAlign(gtk.AlignCenter)
	stepper.SetVAlign(gtk.AlignCenter)
	if dark {
		stepper.SetCSSClasses([]string{"lamha-stroke-stepper", "lamha-stroke-stepper-dark"})
	} else {
		stepper.SetCSSClasses([]string{"lamha-stroke-stepper", "lamha-stroke-stepper-light"})
	}

	minus := gtk.NewButtonWithLabel("−")
	minus.SetTooltipText(withKey(i18n.T("Smaller brush"), keys.WidthDown))
	minus.SetHasFrame(false)
	unfocusable(&minus.Widget)

	valueLabel := gtk.NewLabel(fmt.Sprintf("%.0f", value))
	valueLabel.SetWidthChars(2)
	valueLabel.SetXAlign(0.5)
	valueLabel.SetCSSClasses([]string{"lamha-stroke-value"})
	valueLabel.SetTooltipText(tooltip)

	plus := gtk.NewButtonWithLabel("+")
	plus.SetTooltipText(withKey(i18n.T("Larger brush"), keys.WidthUp))
	plus.SetHasFrame(false)
	unfocusable(&plus.Widget)

	minus.ConnectClicked(func() { adj.SetValue(adj.Value() - 1) })
	plus.ConnectClicked(func() { adj.SetValue(adj.Value() + 1) })

	stepper.Append(minus)
	stepper.Append(valueLabel)
	stepper.Append(plus)
	box.Append(stepper)

	ctrl.stepper = stepper
	ctrl.scale = scale
	ctrl.preview = preview
	ctrl.value = valueLabel
	ctrl.applyLayout()
	adj.ConnectValueChanged(func() {
		ctrl.value.SetText(fmt.Sprintf("%.0f", adj.Value()))
		preview.QueueDraw()
		if ctrl.updating {
			return
		}
		onChange(adj.Value())
	})
	return ctrl
}

func (s *strokeControl) SetVertical(vertical bool) {
	if s == nil {
		return
	}
	s.vertical = vertical
	s.applyLayout()
}

func (s *strokeControl) applyLayout() {
	if s == nil || s.scale == nil {
		return
	}
	width, height := strokePreviewSize(s.compact, s.vertical)
	s.preview.SetHExpand(false)
	s.preview.SetVExpand(false)
	s.preview.SetSizeRequest(width, height)
	s.preview.SetContentWidth(width)
	s.preview.SetContentHeight(height)
	s.preview.QueueDraw()

	if s.vertical {
		s.box.SetOrientation(gtk.OrientationVertical)
		s.box.SetSpacing(2)
		s.box.SetHAlign(gtk.AlignCenter)
		s.box.SetVAlign(gtk.AlignCenter)
		s.box.SetHExpand(false)
		s.box.SetVExpand(false)
		s.scale.SetOrientation(gtk.OrientationVertical)
		s.scale.SetHExpand(false)
		s.scale.SetVExpand(false)
		s.scale.SetSizeRequest(18, 48)
		s.stepper.SetOrientation(gtk.OrientationVertical)
		s.stepper.AddCSSClass("lamha-stroke-stepper-vertical")
		return
	}
	s.box.SetOrientation(gtk.OrientationHorizontal)
	s.box.SetSpacing(6)
	s.box.SetHAlign(gtk.AlignCenter)
	s.box.SetVAlign(gtk.AlignCenter)
	s.box.SetHExpand(false)
	s.box.SetVExpand(false)
	s.scale.SetOrientation(gtk.OrientationHorizontal)
	s.scale.SetHExpand(!s.compact)
	s.scale.SetVExpand(false)
	if s.compact {
		s.scale.SetSizeRequest(72, 18)
	} else {
		s.scale.SetSizeRequest(140, 18)
	}
	s.stepper.SetOrientation(gtk.OrientationHorizontal)
	s.stepper.RemoveCSSClass("lamha-stroke-stepper-vertical")
}

func (s *strokeControl) SetDark(dark bool) {
	if s == nil || s.stepper == nil {
		return
	}
	s.dark = dark
	if dark {
		s.stepper.SetCSSClasses([]string{"lamha-stroke-stepper", "lamha-stroke-stepper-dark"})
	} else {
		s.stepper.SetCSSClasses([]string{"lamha-stroke-stepper", "lamha-stroke-stepper-light"})
	}
	if s.vertical {
		s.stepper.AddCSSClass("lamha-stroke-stepper-vertical")
	}
	if s.preview != nil {
		s.preview.QueueDraw()
	}
}

func (s *strokeControl) SetValue(width float64) {
	if s == nil {
		return
	}
	width = clampStroke(width)
	if s.adjustment.Value() == width {
		return
	}
	s.updating = true
	s.adjustment.SetValue(width)
	s.updating = false
}

func unfocusable(w *gtk.Widget) {
	w.SetFocusable(false)
	w.SetFocusOnClick(false)
}

func strokePreviewSize(compact, vertical bool) (width, height int) {
	if !compact {
		return editorPreviewLength, editorPreviewBreadth
	}
	if vertical {
		return verticalPreviewSize, verticalPreviewSize
	}
	return compactPreviewLength, compactPreviewBreadth
}

func drawStrokePreview(cr *cairo.Context, width, height int, thickness float64, dark, vertical bool) {
	line := clampStroke(thickness)
	if line < 1 {
		line = 1
	}
	room := math.Min(float64(width), float64(height)) - strokePreviewPad*2
	if room < 1 {
		room = 1
	}
	if line > room {
		line = room
	}
	if dark {
		cr.SetSourceRGB(1, 1, 1)
	} else {
		cr.SetSourceRGB(0.17, 0.18, 0.20)
	}
	cr.SetLineCap(cairo.LineCapRound)
	cr.SetLineWidth(line)
	if vertical {
		x := float64(width) / 2
		cr.MoveTo(x, strokePreviewPad)
		cr.LineTo(x, float64(height)-strokePreviewPad)
	} else {
		y := float64(height) / 2
		cr.MoveTo(strokePreviewPad, y)
		cr.LineTo(float64(width)-strokePreviewPad, y)
	}
	cr.Stroke()
}
