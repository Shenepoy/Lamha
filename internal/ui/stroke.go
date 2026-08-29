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
)

type strokeControl struct {
	box        *gtk.Box
	adjustment *gtk.Adjustment
	value      *gtk.Label
	updating   bool
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

	box := gtk.NewBox(gtk.OrientationHorizontal, 8)
	box.SetVAlign(gtk.AlignCenter)
	if !compact {
		label := gtk.NewLabel(i18n.T("Stroke"))
		label.SetCSSClasses([]string{"dim-label"})
		box.Append(label)
	}

	preview := gtk.NewDrawingArea()
	preview.SetVAlign(gtk.AlignCenter)
	preview.SetCSSClasses([]string{"lamha-stroke-preview"})
	preview.SetTooltipText(tooltip)
	preview.SetDrawFunc(func(_ *gtk.DrawingArea, cr *cairo.Context, width, height int) {
		drawStrokePreview(cr, width, height, adj.Value(), dark)
	})
	unfocusable(&preview.Widget)
	resizeStrokePreview(preview, value)
	box.Append(preview)

	scale := gtk.NewScale(gtk.OrientationHorizontal, adj)
	scale.SetDrawValue(false)
	scale.SetVAlign(gtk.AlignCenter)
	scale.SetTooltipText(tooltip)
	if compact {
		scale.SetSizeRequest(88, -1)
	} else {
		scale.SetSizeRequest(140, -1)
		scale.SetHExpand(true)
	}
	unfocusable(&scale.Widget)
	box.Append(scale)

	stepper := gtk.NewBox(gtk.OrientationHorizontal, 0)
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

	ctrl := &strokeControl{box: box, adjustment: adj, value: valueLabel}
	adj.ConnectValueChanged(func() {
		ctrl.value.SetText(fmt.Sprintf("%.0f", adj.Value()))
		resizeStrokePreview(preview, adj.Value())
		preview.QueueDraw()
		if ctrl.updating {
			return
		}
		onChange(adj.Value())
	})
	return ctrl
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

func resizeStrokePreview(preview *gtk.DrawingArea, thickness float64) {
	width, height := strokePreviewSize(thickness)
	preview.SetSizeRequest(width, height)
	preview.SetContentWidth(width)
	preview.SetContentHeight(height)
}

func strokePreviewSize(thickness float64) (width, height int) {
	line := clampStroke(thickness)
	height = int(math.Max(18, line+8))
	width = int(math.Max(64, line*3+16))
	return
}

func drawStrokePreview(cr *cairo.Context, width, height int, thickness float64, dark bool) {
	line := clampStroke(thickness)
	if line < 1 {
		line = 1
	}
	if dark {
		cr.SetSourceRGB(1, 1, 1)
	} else {
		cr.SetSourceRGB(0.17, 0.18, 0.20)
	}
	cr.SetLineCap(cairo.LineCapButt)
	cr.SetLineWidth(line)
	y := float64(height) / 2
	pad := 4.0
	cr.MoveTo(pad, y)
	cr.LineTo(float64(width)-pad, y)
	cr.Stroke()
}
