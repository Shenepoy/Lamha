package ui

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/i18n"
)

func applyAppDirection() {
	if i18n.RTL() {
		gtk.WidgetSetDefaultDirection(gtk.TextDirRTL)
		return
	}
	gtk.WidgetSetDefaultDirection(gtk.TextDirLTR)
}

func applyDirection(widget *gtk.Widget) {
	if i18n.RTL() {
		widget.SetDirection(gtk.TextDirRTL)
		return
	}
	widget.SetDirection(gtk.TextDirLTR)
}

func alignStart(label *gtk.Label) {
	label.SetHAlign(gtk.AlignStart)
	if i18n.RTL() {
		label.SetXAlign(1)
		return
	}
	label.SetXAlign(0)
}

func (w *Window) refreshLocale() {
	applyAppDirection()
	applyDirection(&w.window.Widget)
	w.captureArea.SetLabel(i18n.T("Area"))
	w.captureArea.SetTooltipText(i18n.T("Freeze the screen, then select and mark up an area in Lamha"))
	w.captureWindow.SetLabel(i18n.T("Window"))
	w.captureWindow.SetTooltipText(i18n.T("Window capture is temporarily unavailable"))
	w.captureScreen.SetLabel(i18n.T("Screen"))
	w.captureScreen.SetTooltipText(i18n.T("Freeze the screen, then mark it up in Lamha"))
	if w.menuBtn != nil {
		w.menuBtn.SetTooltipText(i18n.T("Main menu"))
		w.menuBtn.SetMenuModel(w.primaryMenu())
	}
	if w.delayLabel != nil {
		w.delayLabel.SetText(i18n.T("Delay"))
		alignStart(w.delayLabel)
	}
	if w.delay != nil {
		selected := w.delay.Selected()
		w.delay.SetModel(gtk.NewStringList(delayLabels()))
		w.delay.SetSelected(selected)
		w.delay.SetTooltipText(i18n.T("Wait before capture so menus and hover states can appear"))
	}
	if !w.busy {
		w.status.SetText(i18n.T("Ready to capture."))
	}
	alignStart(w.status)
	if w.recentTitle != nil {
		w.recentTitle.SetText(i18n.T("Recent"))
		alignStart(w.recentTitle)
	}
	if w.historyPlaceholder != nil {
		w.historyPlaceholder.SetText(i18n.T("No captures yet"))
	}
	if w.emptyPreview != nil {
		w.emptyPreview.SetText(i18n.T("Select a capture to preview it."))
	}
	if w.copyButton != nil {
		w.copyButton.SetLabel(i18n.T("Copy image"))
	}
	if w.annotateBtn != nil {
		w.annotateBtn.SetLabel(i18n.T("Annotate"))
		w.annotateBtn.SetTooltipText(i18n.T("Open the markup tools on this capture"))
	}
	if w.openButton != nil {
		w.openButton.SetLabel(i18n.T("Open captures folder"))
	}
	w.rebuildSettings()
	w.rebuildAbout()
	path := ""
	if w.lastCapture != nil {
		path = w.lastCapture.Path
	}
	w.refreshHistory(path)
}
