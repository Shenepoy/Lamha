package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/annotate"
	"github.com/lamha-app/lamha/internal/i18n"
)

type stepNumberPop struct {
	pop   *gtk.Popover
	label *gtk.Label
	index int
	apply func(index, delta int)
}

func newStepNumberPop(apply func(index, delta int)) *stepNumberPop {
	return &stepNumberPop{index: -1, apply: apply}
}

func stepAnchor(stroke annotate.Stroke) annotate.Point {
	if len(stroke.Points) > 0 {
		return stroke.Points[0]
	}
	return annotate.Point{X: stroke.X1, Y: stroke.Y1}
}

func (p *stepNumberPop) ensure(parent gtk.Widgetter) {
	if p == nil || p.pop != nil || parent == nil {
		return
	}
	ensureToolbarCSS()

	stepper := gtk.NewBox(gtk.OrientationHorizontal, 0)
	if themePrefersDark() {
		stepper.SetCSSClasses([]string{"lamha-stroke-stepper", "lamha-stroke-stepper-dark"})
	} else {
		stepper.SetCSSClasses([]string{"lamha-stroke-stepper", "lamha-stroke-stepper-light"})
	}

	minus := gtk.NewButtonWithLabel("−")
	minus.SetTooltipText(i18n.T("Decrease step number"))
	minus.SetHasFrame(false)
	unfocusable(&minus.Widget)

	p.label = gtk.NewLabel("1")
	p.label.SetWidthChars(2)
	p.label.SetXAlign(0.5)
	p.label.SetCSSClasses([]string{"lamha-stroke-value"})
	p.label.SetTooltipText(i18n.T("Step number"))

	plus := gtk.NewButtonWithLabel("+")
	plus.SetTooltipText(i18n.T("Increase step number"))
	plus.SetHasFrame(false)
	unfocusable(&plus.Widget)

	minus.ConnectClicked(func() {
		if p.apply != nil && p.index >= 0 {
			p.apply(p.index, -1)
		}
	})
	plus.ConnectClicked(func() {
		if p.apply != nil && p.index >= 0 {
			p.apply(p.index, 1)
		}
	})

	stepper.Append(minus)
	stepper.Append(p.label)
	stepper.Append(plus)

	pop := gtk.NewPopover()
	pop.SetParent(parent)
	pop.SetHasArrow(true)
	pop.SetAutohide(true)
	pop.SetPosition(gtk.PosRight)
	pop.AddCSSClass("lamha-step-pop")
	pop.SetChild(stepper)
	p.pop = pop
}

func (p *stepNumberPop) present(parent gtk.Widgetter, view annotate.View, stroke annotate.Stroke, index int) {
	if p == nil || stroke.Tool != annotate.ToolStep {
		return
	}
	p.ensure(parent)
	if p.pop == nil || p.label == nil {
		return
	}
	p.index = index
	p.label.SetText(fmt.Sprintf("%d", stroke.Step))
	pt := stepAnchor(stroke)
	wx, wy := view.ToWidget(pt.X, pt.Y)
	r := annotate.StepDiameter(stroke.Width) * view.Scale / 2
	if r < 12 {
		r = 12
	}
	rect := gdk.NewRectangle(int(wx-r), int(wy-r), int(r*2), int(r*2))
	p.pop.SetPointingTo(&rect)
	p.pop.Popup()
}

func (p *stepNumberPop) hide() {
	if p == nil || p.pop == nil {
		return
	}
	p.index = -1
	p.pop.Popdown()
}
