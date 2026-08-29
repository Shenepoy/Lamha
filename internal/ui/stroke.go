package ui

import (
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
	preview.SetSizeRequest(36, 22)
	preview.SetContentWidth(36)
	preview.SetContentHeight(22)
	preview.SetCSSClasses([]string{"lamha-stroke-preview"})
	preview.SetTooltipText(tooltip)
	preview.SetDrawFunc(func(_ *gtk.DrawingArea, cr *cairo.Context, width, height int) {
		drawStrokePreview(cr, width, height, adj.Value(), dark)
	})
	box.Append(preview)

	scale := gtk.NewScale(gtk.OrientationHorizontal, adj)
	scale.SetDrawValue(false)
	scale.SetTooltipText(tooltip)
	if compact {
		scale.SetSizeRequest(88, -1)
	} else {
		scale.SetSizeRequest(140, -1)
		scale.SetHExpand(true)
	}
	box.Append(scale)

	spin := gtk.NewSpinButton(adj, 1, 0)
	spin.SetNumeric(true)
	spin.SetSnapToTicks(true)
	spin.SetSizeRequest(56, -1)
	spin.SetTooltipText(tooltip)
	box.Append(spin)

	ctrl := &strokeControl{box: box, adjustment: adj}
	adj.ConnectValueChanged(func() {
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

func drawStrokePreview(cr *cairo.Context, width, height int, thickness float64, dark bool) {
	line := math.Min(clampStroke(thickness), float64(height)-4)
	if line < 1 {
		line = 1
	}
	if dark {
		cr.SetSourceRGB(1, 1, 1)
	} else {
		cr.SetSourceRGB(0.17, 0.18, 0.20)
	}
	cr.SetLineCap(cairo.LineCapRound)
	cr.SetLineWidth(line)
	y := float64(height) / 2
	pad := math.Max(line/2+1, 4)
	cr.MoveTo(pad, y)
	cr.LineTo(float64(width)-pad, y)
	cr.Stroke()
}
