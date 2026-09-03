package ui

import (
	"log"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/brand"
	"github.com/lamha-app/lamha/internal/crash"
	"github.com/lamha-app/lamha/internal/i18n"
)

// ShowCrashNotice explains an unexpected previous exit and offers explicit
// actions for reviewing or reporting the local diagnostics.
func (w *Window) ShowCrashNotice(report crash.Report, reviewed func()) {
	if w == nil || w.window == nil || report.Text == "" {
		return
	}

	win := gtk.NewWindow()
	if app := w.window.Application(); app != nil {
		win.SetApplication(app)
	}
	win.SetTitle(i18n.T("Lamha crash report"))
	win.SetIconName(brand.Name)
	applyDirection(&win.Widget)
	win.SetTransientFor(&w.window.Window)
	win.SetModal(true)
	win.SetDefaultSize(560, 360)

	closed := false
	finish := func(clear bool) {
		if closed {
			return
		}
		closed = true
		if clear && reviewed != nil {
			reviewed()
		}
		win.Destroy()
	}
	win.ConnectCloseRequest(func() bool {
		finish(true)
		return true
	})

	root := gtk.NewBox(gtk.OrientationVertical, 12)
	root.SetMarginTop(24)
	root.SetMarginBottom(24)
	root.SetMarginStart(24)
	root.SetMarginEnd(24)

	heading := gtk.NewLabel(i18n.T("Lamha stopped unexpectedly"))
	heading.SetCSSClasses([]string{"title-3", "heading"})
	alignStart(heading)
	root.Append(heading)

	detail := gtk.NewLabel(i18n.T("Lamha detected that its previous process did not shut down normally. The report below is kept locally until you review it."))
	detail.SetWrap(true)
	detail.SetMaxWidthChars(72)
	alignStart(detail)
	root.Append(detail)

	path := gtk.NewLabel(i18n.Tf("Report saved at:\n%s", report.Path))
	path.SetWrap(true)
	path.SetSelectable(true)
	path.SetCSSClasses([]string{"dim-label", "dimmed", "monospace"})
	alignStart(path)
	root.Append(path)

	privacy := gtk.NewLabel(i18n.T("Review the report before sharing. It may contain file paths and other details from the app log."))
	privacy.SetWrap(true)
	privacy.SetMaxWidthChars(72)
	privacy.SetCSSClasses([]string{"dim-label", "dimmed"})
	alignStart(privacy)
	root.Append(privacy)

	actions := gtk.NewBox(gtk.OrientationHorizontal, 8)
	actions.SetHAlign(gtk.AlignEnd)

	open := gtk.NewButtonWithLabel(i18n.T("Open report"))
	open.ConnectClicked(func() {
		if err := gio.AppInfoLaunchDefaultForURI(fileURI(report.Path), nil); err != nil {
			log.Printf("could not open crash report: %v", err)
			w.status.SetText(i18n.Tf("Could not open crash report: %v", err))
		}
	})
	actions.Append(open)

	copy := gtk.NewButtonWithLabel(i18n.T("Copy report"))
	copy.ConnectClicked(func() {
		if err := copyText(report.Text); err != nil {
			log.Printf("could not copy crash report: %v", err)
			w.status.SetText(i18n.Tf("Could not copy crash report: %v", err))
			return
		}
		w.status.SetText(i18n.T("Crash report copied to the clipboard."))
		finish(true)
	})
	actions.Append(copy)

	reportButton := gtk.NewButtonWithLabel(i18n.T("Report on GitHub"))
	reportButton.SetCSSClasses([]string{"suggested-action"})
	reportButton.ConnectClicked(func() {
		issueURL := crash.IssueURL(brand.IssuesURL, report)
		if issueURL == "" {
			w.status.SetText(i18n.T("Could not create a GitHub issue link."))
			return
		}
		if err := gio.AppInfoLaunchDefaultForURI(issueURL, nil); err != nil {
			log.Printf("could not open GitHub issue form: %v", err)
			w.status.SetText(i18n.Tf("Could not open GitHub issue form: %v", err))
			return
		}
		finish(true)
	})
	actions.Append(reportButton)

	dismiss := gtk.NewButtonWithLabel(i18n.T("Dismiss"))
	dismiss.ConnectClicked(func() { finish(true) })
	actions.Append(dismiss)
	root.Append(actions)

	win.SetChild(root)
	win.Present()
}
