package ui

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/brand"
	"github.com/lamha-app/lamha/internal/i18n"
	"github.com/lamha-app/lamha/internal/prefs"
)

func (w *Window) openSettings() {
	win := gtk.NewWindow()
	if app := w.window.Application(); app != nil {
		win.SetApplication(app)
	}
	win.SetTitle(i18n.T("Settings"))
	win.SetIconName(brand.Name)
	applyDirection(&win.Widget)
	win.SetTransientFor(&w.window.Window)
	win.SetModal(true)
	win.SetDefaultSize(420, 220)
	win.SetHideOnClose(true)

	header := gtk.NewHeaderBar()
	win.SetTitlebar(header)

	zoomLabel := gtk.NewLabel(zoomCaption(prefs.Current().MagnifierZoom()))
	alignStart(zoomLabel)
	zoomLabel.SetCSSClasses([]string{"title-4"})

	help := gtk.NewLabel(i18n.T("How much the capture lens enlarges pixels under the pointer."))
	alignStart(help)
	help.SetWrap(true)
	help.SetCSSClasses([]string{"dim-label"})

	scale := gtk.NewScaleWithRange(gtk.OrientationHorizontal, prefs.MinMagnifierZoom, prefs.MaxMagnifierZoom, 0.5)
	scale.SetDrawValue(false)
	scale.SetHExpand(true)
	scale.SetValue(prefs.Current().MagnifierZoom())
	scale.AddMark(prefs.DefaultMagnifierZoom, gtk.PosBottom, i18n.T("Default"))
	scale.ConnectValueChanged(func() {
		zoom := prefs.ClampZoom(scale.Value())
		if err := prefs.Current().SetMagnifierZoom(zoom); err != nil {
			w.status.SetText(i18n.Tf("Could not save settings: %v", err))
			return
		}
		zoomLabel.SetText(zoomCaption(zoom))
	})

	box := gtk.NewBox(gtk.OrientationVertical, 10)
	box.SetMarginTop(18)
	box.SetMarginBottom(18)
	box.SetMarginStart(18)
	box.SetMarginEnd(18)
	box.Append(zoomLabel)
	box.Append(help)
	box.Append(scale)
	win.SetChild(box)
	win.Present()
}

func zoomCaption(zoom float64) string {
	return i18n.Tf("Magnifier zoom: %.1f×", prefs.ClampZoom(zoom))
}
