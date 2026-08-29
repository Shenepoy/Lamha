package ui

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/brand"
	"github.com/lamha-app/lamha/internal/i18n"
)

func (w *Window) openAbout() {
	if w.aboutWin != nil {
		w.aboutWin.Present()
		return
	}

	win := gtk.NewWindow()
	if app := w.window.Application(); app != nil {
		win.SetApplication(app)
	}
	win.SetTitle(i18n.T("About Me"))
	win.SetIconName(brand.Name)
	applyDirection(&win.Widget)
	win.SetTransientFor(&w.window.Window)
	win.SetModal(true)
	win.SetDefaultSize(400, 360)
	win.SetHideOnClose(true)
	win.ConnectCloseRequest(func() bool {
		w.aboutWin = nil
		win.Destroy()
		return true
	})

	header := gtk.NewHeaderBar()
	win.SetTitlebar(header)
	win.SetChild(aboutWindowChild())
	w.aboutWin = win
	win.Present()
}

func (w *Window) rebuildAbout() {
	if w.aboutWin == nil {
		return
	}
	w.aboutWin.SetTitle(i18n.T("About Me"))
	applyDirection(&w.aboutWin.Widget)
	w.aboutWin.SetChild(aboutWindowChild())
}

func aboutWindowChild() *gtk.Box {
	box := aboutContent(true)
	box.SetMarginTop(24)
	box.SetMarginBottom(24)
	box.SetMarginStart(24)
	box.SetMarginEnd(24)
	return box
}

func aboutContent(centered bool) *gtk.Box {
	name := gtk.NewLabel(brand.DeveloperName)
	name.SetWrap(true)
	name.SetCSSClasses([]string{"title-3"})

	summary := gtk.NewLabel(i18n.T("I am a researcher and developer skilled in Dart, Rust, and AI. I build diverse projects, from mobile apps to server tools, while exploring 3D printing, virtual cycling, and mastering multiple languages."))
	summary.SetWrap(true)
	summary.SetMaxWidthChars(42)

	profile := gtk.NewLinkButtonWithLabel(brand.DeveloperProfileURL, i18n.T("View profile"))
	source := gtk.NewLinkButtonWithLabel(brand.SourceURL, i18n.T("Source Code"))
	source.SetTooltipText(i18n.T("Browse the app source on GitHub"))
	updates := gtk.NewLinkButtonWithLabel(brand.UpdateURL, i18n.T("Updates"))
	updates.SetTooltipText(i18n.T("Check for updates on GitHub"))

	if centered {
		name.SetHAlign(gtk.AlignCenter)
		name.SetXAlign(0.5)
		name.SetJustify(gtk.JustifyCenter)
		summary.SetHAlign(gtk.AlignCenter)
		summary.SetXAlign(0.5)
		summary.SetJustify(gtk.JustifyCenter)
		profile.SetHAlign(gtk.AlignCenter)
		source.SetHAlign(gtk.AlignCenter)
		updates.SetHAlign(gtk.AlignCenter)
	} else {
		alignStart(name)
		alignStart(summary)
	}

	box := gtk.NewBox(gtk.OrientationVertical, 10)
	box.Append(name)
	box.Append(summary)
	box.Append(profile)
	box.Append(source)
	box.Append(updates)
	return box
}
