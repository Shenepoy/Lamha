package ui

import (
	"math"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/annotate"
	"github.com/lamha-app/lamha/internal/i18n"
)

type iconDraw func(cr *cairo.Context, w, h float64)

func newIconCanvas(draw iconDraw) *gtk.DrawingArea {
	area := gtk.NewDrawingArea()
	area.SetSizeRequest(28, 28)
	area.SetContentWidth(28)
	area.SetContentHeight(28)
	area.SetDrawFunc(func(_ *gtk.DrawingArea, cr *cairo.Context, width, height int) {
		cr.SetSourceRGB(1, 1, 1)
		cr.SetLineWidth(1.7)
		cr.SetLineCap(cairo.LineCapRound)
		cr.SetLineJoin(cairo.LineJoinRound)
		draw(cr, float64(width), float64(height))
	})
	return area
}

func newDrawnToggle(draw iconDraw, tip string) *gtk.ToggleButton {
	button := gtk.NewToggleButton()
	button.SetChild(newIconCanvas(draw))
	button.SetTooltipText(tip)
	button.SetHasFrame(false)
	return button
}

func iconBox(w, h float64) (x, y, s float64) {
	pad := math.Min(w, h) * 0.18
	return pad, pad, math.Min(w, h) - 2*pad
}

func iconSelect(cr *cairo.Context, w, h float64) {
	x, y, s := iconBox(w, h)
	cr.SetDash([]float64{3.2, 2.4}, 0)
	cr.Rectangle(x+0.5, y+0.5, s-1, s-1)
	cr.Stroke()
	cr.SetDash(nil, 0)
}

func iconMove(cr *cairo.Context, w, h float64) {
	cx, cy := w/2, h/2
	_, _, s := iconBox(w, h)
	arm := s * 0.46
	head := s * 0.18
	cr.MoveTo(cx-arm, cy)
	cr.LineTo(cx+arm, cy)
	cr.MoveTo(cx, cy-arm)
	cr.LineTo(cx, cy+arm)
	cr.Stroke()
	arrow(cr, cx, cy-arm, 0, -1, head)
	arrow(cr, cx, cy+arm, 0, 1, head)
	arrow(cr, cx-arm, cy, -1, 0, head)
	arrow(cr, cx+arm, cy, 1, 0, head)
}

func arrow(cr *cairo.Context, x, y, dx, dy, size float64) {
	px, py := -dy, dx
	cr.MoveTo(x, y)
	cr.LineTo(x-dx*size+px*size*0.55, y-dy*size+py*size*0.55)
	cr.LineTo(x-dx*size-px*size*0.55, y-dy*size-py*size*0.55)
	cr.ClosePath()
	cr.Fill()
}

func iconPen(cr *cairo.Context, w, h float64) {
	x, y, s := iconBox(w, h)
	cr.MoveTo(x+s*0.18, y+s*0.82)
	cr.LineTo(x+s*0.72, y+s*0.28)
	cr.LineTo(x+s*0.84, y+s*0.16)
	cr.LineTo(x+s*0.62, y+s*0.28)
	cr.ClosePath()
	cr.Stroke()
	cr.MoveTo(x+s*0.18, y+s*0.82)
	cr.LineTo(x+s*0.32, y+s*0.68)
	cr.Stroke()
	cr.Rectangle(x+s*0.12, y+s*0.78, s*0.16, s*0.16)
	cr.Fill()
}

func iconArrow(cr *cairo.Context, w, h float64) {
	x, y, s := iconBox(w, h)
	cr.MoveTo(x+s*0.12, y+s*0.78)
	cr.LineTo(x+s*0.62, y+s*0.28)
	cr.Stroke()
	arrow(cr, x+s*0.78, y+s*0.16, 0.75, -0.66, s*0.28)
}

func iconRect(cr *cairo.Context, w, h float64) {
	x, y, s := iconBox(w, h)
	cr.Rectangle(x+0.5, y+s*0.12, s-1, s*0.76)
	cr.Stroke()
}

func iconEllipse(cr *cairo.Context, w, h float64) {
	x, y, s := iconBox(w, h)
	cr.Save()
	cr.Translate(x+s/2, y+s/2)
	cr.Scale(s*0.48, s*0.36)
	cr.Arc(0, 0, 1, 0, 2*math.Pi)
	cr.Restore()
	cr.Stroke()
}

func iconHighlight(cr *cairo.Context, w, h float64) {
	x, y, s := iconBox(w, h)
	cr.SetLineWidth(s * 0.28)
	cr.MoveTo(x+s*0.08, y+s*0.72)
	cr.LineTo(x+s*0.72, y+s*0.18)
	cr.Stroke()
	cr.SetLineWidth(1.7)
	cr.MoveTo(x+s*0.74, y+s*0.08)
	cr.LineTo(x+s*0.92, y+s*0.22)
	cr.LineTo(x+s*0.78, y+s*0.28)
	cr.ClosePath()
	cr.Fill()
}

func iconBlur(cr *cairo.Context, w, h float64) {
	x, y, s := iconBox(w, h)
	cell := s / 3
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			shade := 0.45 + 0.18*float64((row+col)%3)
			cr.SetSourceRGB(shade, shade, shade)
			cr.Rectangle(x+float64(col)*cell+0.6, y+float64(row)*cell+0.6, cell-1.2, cell-1.2)
			cr.Fill()
		}
	}
	cr.SetSourceRGB(1, 1, 1)
	cr.Rectangle(x+0.5, y+0.5, s-1, s-1)
	cr.Stroke()
}

func iconSteps(cr *cairo.Context, w, h float64) {
	x, y, s := iconBox(w, h)
	cr.Arc(x+s/2, y+s/2, s*0.46, 0, 2*math.Pi)
	cr.Stroke()
	cr.SelectFontFace("sans-serif", cairo.FontSlantNormal, cairo.FontWeightBold)
	cr.SetFontSize(s * 0.62)
	cr.MoveTo(x+s*0.32, y+s*0.72)
	cr.ShowText("1")
}

func iconText(cr *cairo.Context, w, h float64) {
	x, y, s := iconBox(w, h)
	glyph := "A"
	if i18n.Arabic() {
		glyph = "أ"
	}
	cr.SelectFontFace("sans-serif", cairo.FontSlantNormal, cairo.FontWeightBold)
	cr.SetFontSize(s * 0.92)
	cr.MoveTo(x+s*0.12, y+s*0.82)
	cr.ShowText(glyph)
}

func iconEraser(cr *cairo.Context, w, h float64) {
	x, y, s := iconBox(w, h)
	cr.MoveTo(x+s*0.18, y+s*0.42)
	cr.LineTo(x+s*0.52, y+s*0.08)
	cr.LineTo(x+s*0.86, y+s*0.42)
	cr.LineTo(x+s*0.52, y+s*0.76)
	cr.ClosePath()
	cr.Stroke()
	cr.MoveTo(x+s*0.18, y+s*0.42)
	cr.LineTo(x+s*0.86, y+s*0.42)
	cr.Stroke()
	cr.MoveTo(x+s*0.22, y+s*0.78)
	cr.LineTo(x+s*0.58, y+s*0.78)
	cr.Stroke()
}

func iconAreaErase(cr *cairo.Context, w, h float64) {
	x, y, s := iconBox(w, h)
	cr.SetDash([]float64{2.6, 2.0}, 0)
	cr.Rectangle(x+0.5, y+0.5, s*0.72, s*0.72)
	cr.Stroke()
	cr.SetDash(nil, 0)
	cr.MoveTo(x+s*0.42, y+s*0.58)
	cr.LineTo(x+s*0.62, y+s*0.38)
	cr.LineTo(x+s*0.86, y+s*0.62)
	cr.LineTo(x+s*0.66, y+s*0.82)
	cr.ClosePath()
	cr.Fill()
}

func toolIcon(tool annotate.Tool) iconDraw {
	switch tool {
	case annotate.ToolSelect:
		return iconSelect
	case annotate.ToolMove:
		return iconMove
	case annotate.ToolPen:
		return iconPen
	case annotate.ToolArrow:
		return iconArrow
	case annotate.ToolBox:
		return iconRect
	case annotate.ToolEllipse:
		return iconEllipse
	case annotate.ToolHighlight:
		return iconHighlight
	case annotate.ToolBlur:
		return iconBlur
	case annotate.ToolStep:
		return iconSteps
	case annotate.ToolText:
		return iconText
	case annotate.ToolMagicErase:
		return iconEraser
	case annotate.ToolAreaErase:
		return iconAreaErase
	default:
		return iconPen
	}
}
