package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"

	"github.com/lamha-app/lamha/internal/capture"
	"github.com/lamha-app/lamha/internal/i18n"
)

func (w *Window) newHistoryRow(item capture.SavedCapture) *gtk.ListBoxRow {
	row := gtk.NewListBoxRow()

	box := gtk.NewBox(gtk.OrientationHorizontal, 12)
	box.SetMarginTop(8)
	box.SetMarginBottom(8)
	box.SetMarginStart(10)
	box.SetMarginEnd(10)

	thumb := gtk.NewPicture()
	thumb.SetFilename(item.Path)
	thumb.SetSizeRequest(80, 50)
	thumb.SetCanShrink(true)
	thumb.SetContentFit(gtk.ContentFitCover)
	thumb.SetHAlign(gtk.AlignStart)
	thumb.SetVAlign(gtk.AlignCenter)
	thumb.SetCSSClasses([]string{"card"})
	box.Append(thumb)

	text := gtk.NewBox(gtk.OrientationVertical, 2)
	text.SetHExpand(true)
	text.SetVAlign(gtk.AlignCenter)

	when := gtk.NewLabel(formatCaptureTime(item.CreatedAt, time.Now()))
	alignStart(when)
	when.SetEllipsize(pango.EllipsizeEnd)
	text.Append(when)

	meta := gtk.NewLabel(formatCaptureMeta(item))
	alignStart(meta)
	meta.SetCSSClasses([]string{"dim-label", "caption"})
	meta.SetEllipsize(pango.EllipsizeEnd)
	text.Append(meta)

	box.Append(text)
	row.SetChild(box)
	w.attachHistoryMenu(row, item)
	return row
}

func (w *Window) attachHistoryMenu(row *gtk.ListBoxRow, item capture.SavedCapture) {
	click := gtk.NewGestureClick()
	click.SetButton(gdk.BUTTON_SECONDARY)
	click.SetPropagationPhase(gtk.PhaseCapture)
	click.ConnectPressed(func(nPress int, x, y float64) {
		click.SetState(gtk.EventSequenceClaimed)
	})
	click.ConnectReleased(func(nPress int, x, y float64) {
		w.popupCaptureMenu(&row.Widget, item)
	})
	row.AddController(click)
}

func (w *Window) attachPreviewMenu() {
	click := gtk.NewGestureClick()
	click.SetButton(gdk.BUTTON_SECONDARY)
	click.SetPropagationPhase(gtk.PhaseCapture)
	click.ConnectPressed(func(nPress int, x, y float64) {
		if w.lastCapture == nil {
			return
		}
		click.SetState(gtk.EventSequenceClaimed)
	})
	click.ConnectReleased(func(nPress int, x, y float64) {
		if w.lastCapture == nil {
			return
		}
		w.popupCaptureMenu(&w.previewStack.Widget, *w.lastCapture)
	})
	w.previewStack.AddController(click)
}

func (w *Window) popupCaptureMenu(parent gtk.Widgetter, item capture.SavedCapture) {
	if w.historyMenu != nil {
		w.historyMenu.Popdown()
		w.historyMenu.Unparent()
		w.historyMenu = nil
	}

	target := item
	w.menuTarget = &target

	edit := gio.NewMenu()
	edit.Append(i18n.T("Annotate"), "win.annotate-capture")
	edit.Append(i18n.T("Copy image"), "win.copy-capture")

	clip := gio.NewMenu()
	clip.Append(i18n.T("Copy path"), "win.copy-path")
	clip.Append(i18n.T("Copy file name"), "win.copy-name")

	open := gio.NewMenu()
	open.Append(i18n.T("Open"), "win.open-capture")
	open.Append(i18n.T("Show in folder"), "win.open-folder")

	danger := gio.NewMenu()
	danger.Append(i18n.T("Delete"), "win.delete-capture")

	model := gio.NewMenu()
	model.AppendSection("", edit)
	model.AppendSection("", clip)
	model.AppendSection("", open)
	model.AppendSection("", danger)

	pop := gtk.NewPopoverMenuFromModel(model)
	pop.SetParent(parent)
	pop.SetHasArrow(false)
	pop.InsertActionGroup("win", &w.window.ActionGroup)
	w.historyMenu = pop
	pop.ConnectClosed(func() {
		if w.historyMenu == pop {
			w.historyMenu = nil
		}
	})
	pop.Popup()
}

func (w *Window) contextCapture() (capture.SavedCapture, bool) {
	if w.menuTarget != nil {
		return *w.menuTarget, true
	}
	return capture.SavedCapture{}, false
}

func (w *Window) annotateSelected() {
	item, ok := w.contextCapture()
	if !ok {
		return
	}
	w.openEditor(item.Path)
}

func (w *Window) copyContextImage() {
	item, ok := w.contextCapture()
	if !ok {
		return
	}
	if err := copyImageFile(item.Path); err != nil {
		w.status.SetText(i18n.Tf("Could not copy image: %v", err))
		return
	}
	w.status.SetText(i18n.T("Image copied to clipboard."))
}

func (w *Window) copySelectedPath() {
	item, ok := w.contextCapture()
	if !ok {
		return
	}
	w.copyCaptureText(item.Path, i18n.T("Path copied to clipboard."))
}

func (w *Window) copySelectedName() {
	item, ok := w.contextCapture()
	if !ok {
		return
	}
	w.copyCaptureText(filepath.Base(item.Path), i18n.T("File name copied to clipboard."))
}

func (w *Window) openSelectedFile() {
	item, ok := w.contextCapture()
	if !ok {
		return
	}
	w.openCaptureFile(item)
}

func (w *Window) deleteSelected() {
	item, ok := w.contextCapture()
	if !ok {
		return
	}
	w.confirmDeleteCapture(item)
}

func (w *Window) copyCaptureText(text, ok string) {
	if err := copyText(text); err != nil {
		w.status.SetText(i18n.Tf("Could not copy: %v", err))
		return
	}
	w.status.SetText(ok)
}

func (w *Window) openCaptureFile(item capture.SavedCapture) {
	if err := gio.AppInfoLaunchDefaultForURI(fileURI(item.Path), nil); err != nil {
		w.status.SetText(i18n.Tf("Could not open capture: %v", err))
	}
}

func (w *Window) confirmDeleteCapture(item capture.SavedCapture) {
	win := gtk.NewWindow()
	if app := w.window.Application(); app != nil {
		win.SetApplication(app)
	}
	win.SetTransientFor(&w.window.Window)
	win.SetModal(true)
	win.SetTitle(i18n.T("Delete capture"))
	win.SetDefaultSize(380, 160)

	header := gtk.NewHeaderBar()
	win.SetTitlebar(header)

	body := gtk.NewLabel(i18n.T("Delete this capture from disk? This cannot be undone."))
	body.SetWrap(true)
	body.SetMarginTop(16)
	body.SetMarginBottom(8)
	body.SetMarginStart(18)
	body.SetMarginEnd(18)

	cancel := gtk.NewButtonWithLabel(i18n.T("Cancel"))
	cancel.ConnectClicked(func() { win.Destroy() })
	remove := gtk.NewButtonWithLabel(i18n.T("Delete"))
	remove.SetCSSClasses([]string{"destructive-action"})
	remove.ConnectClicked(func() {
		win.Destroy()
		w.deleteCapture(item)
	})

	actions := gtk.NewBox(gtk.OrientationHorizontal, 8)
	actions.SetHAlign(gtk.AlignEnd)
	actions.SetMarginTop(8)
	actions.SetMarginBottom(16)
	actions.SetMarginStart(18)
	actions.SetMarginEnd(18)
	actions.Append(cancel)
	actions.Append(remove)

	root := gtk.NewBox(gtk.OrientationVertical, 0)
	root.Append(body)
	root.Append(actions)
	win.SetChild(root)
	win.Present()
}

func (w *Window) deleteCapture(item capture.SavedCapture) {
	if err := os.Remove(item.Path); err != nil {
		w.status.SetText(i18n.Tf("Could not delete capture: %v", err))
		return
	}
	w.status.SetText(i18n.T("Capture deleted."))
	next := ""
	if w.lastCapture != nil && w.lastCapture.Path != item.Path {
		next = w.lastCapture.Path
	}
	w.refreshHistory(next)
}

func formatCaptureMeta(item capture.SavedCapture) string {
	size := formatFileSize(item.Bytes)
	if item.Width > 0 && item.Height > 0 {
		return fmt.Sprintf("%d×%d • %s", item.Width, item.Height, size)
	}
	return size
}

func formatFileSize(n int64) string {
	switch {
	case n < 1024:
		return i18n.Tf("%d B", n)
	case n < 1024*1024:
		return i18n.Tf("%.1f KB", float64(n)/1024)
	default:
		return i18n.Tf("%.1f MB", float64(n)/(1024*1024))
	}
}

func formatCaptureTime(when, now time.Time) string {
	when = when.In(now.Location())
	elapsed := now.Sub(when)
	switch {
	case elapsed < time.Minute:
		return i18n.T("Just now")
	case elapsed < time.Hour:
		return i18n.Tf("%d minutes ago", int(elapsed.Minutes()))
	case elapsed < 24*time.Hour:
		return i18n.Tf("%d hours ago", int(elapsed.Hours()))
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	day := time.Date(when.Year(), when.Month(), when.Day(), 0, 0, 0, 0, now.Location())
	days := int(today.Sub(day) / (24 * time.Hour))
	switch {
	case days == 1:
		return i18n.T("Yesterday")
	case days > 1 && days < 7:
		return i18n.Tf("%d days ago", days)
	default:
		return when.Format("2006-01-02")
	}
}
