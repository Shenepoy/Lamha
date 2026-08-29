package ui

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/autostart"
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
	win.SetDefaultSize(420, 300)
	win.SetHideOnClose(true)

	header := gtk.NewHeaderBar()
	win.SetTitlebar(header)

	zoomTitle := gtk.NewLabel(i18n.T("Capture"))
	alignStart(zoomTitle)
	zoomTitle.SetCSSClasses([]string{"title-4"})

	zoomLabel := gtk.NewLabel(zoomCaption(prefs.Current().MagnifierZoom()))
	alignStart(zoomLabel)

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

	startup := gtk.NewLabel(i18n.T("Startup"))
	alignStart(startup)
	startup.SetCSSClasses([]string{"title-4"})
	startup.SetMarginTop(8)

	auto := gtk.NewCheckButtonWithLabel(i18n.T("Start in the background on login"))
	auto.SetActive(autostart.Enabled())
	auto.ConnectToggled(func() {
		if err := autostart.SetEnabled(auto.Active()); err != nil {
			w.status.SetText(i18n.Tf("Could not update login start: %v", err))
		}
	})

	box := gtk.NewBox(gtk.OrientationVertical, 10)
	box.SetMarginTop(18)
	box.SetMarginBottom(18)
	box.SetMarginStart(18)
	box.SetMarginEnd(18)
	box.Append(zoomTitle)
	box.Append(zoomLabel)
	box.Append(help)
	box.Append(scale)
	box.Append(startup)
	box.Append(auto)
	win.SetChild(box)
	win.Present()
}

func zoomCaption(zoom float64) string {
	return i18n.Tf("Magnifier zoom: %.1f×", prefs.ClampZoom(zoom))
}
