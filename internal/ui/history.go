package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"

	"github.com/lamha-app/lamha/internal/capture"
	"github.com/lamha-app/lamha/internal/i18n"
)

var contextMenuCSSOnce sync.Once

func ensureContextMenuCSS() {
	contextMenuCSSOnce.Do(func() {
		provider := gtk.NewCSSProvider()
		provider.LoadFromData(`
popover.lamha-context-menu contents,
popover.lamha-context-menu scrolledwindow,
popover.lamha-context-menu scrolledwindow > viewport {
  max-height: none;
}
`)
		if display := gdk.DisplayGetDefault(); display != nil {
			gtk.StyleContextAddProviderForDisplay(display, provider, gtk.STYLE_PROVIDER_PRIORITY_USER)
		}
	})
}

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
	meta.SetCSSClasses([]string{"dim-label", "dimmed", "caption"})
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
		w.popupCaptureMenu(&row.Widget, item, x, y)
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
		w.popupCaptureMenu(&w.previewStack.Widget, *w.lastCapture, x, y)
	})
	w.previewStack.AddController(click)
}

func (w *Window) popupCaptureMenu(parent gtk.Widgetter, item capture.SavedCapture, x, y float64) {
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

	host := w.menuHost()
	hx, hy := x, y
	if src := gtk.BaseWidget(parent); src != nil && host != nil {
		if px, py, ok := widgetPoint(src, host, x, y); ok {
			hx, hy = px, py
		}
	}
	if host == nil {
		host = parent
	}

	ensureContextMenuCSS()
	pop := gtk.NewPopoverMenuFromModel(model)
	pop.SetParent(host)
	pop.AddCSSClass("lamha-context-menu")
	pop.SetHasArrow(false)
	pop.SetHAlign(gtk.AlignStart)
	if hostH := widgetHeight(host); hostH > 0 && hy > float64(hostH)/2 {
		pop.SetPosition(gtk.PosTop)
	} else {
		pop.SetPosition(gtk.PosBottom)
	}
	rect := gdk.NewRectangle(int(hx), int(hy), 1, 1)
	pop.SetPointingTo(&rect)
	pop.InsertActionGroup("win", &w.window.ActionGroup)
	w.historyMenu = pop
	pop.ConnectClosed(func() {
		if w.historyMenu == pop {
			w.historyMenu = nil
		}
	})
	maxH := menuMaxHeight(host)
	unclipPopoverMenu(pop, maxH)
	pop.Popup()
	pop.Present()
	glib.IdleAdd(func() bool {
		if w.historyMenu != pop {
			return false
		}
		unclipPopoverMenu(pop, maxH)
		pop.Present()
		return false
	})
}

func (w *Window) menuHost() gtk.Widgetter {
	if w.window == nil {
		return nil
	}
	if child := w.window.Child(); child != nil {
		return child
	}
	return &w.window.Widget
}

func widgetHeight(widget gtk.Widgetter) int {
	if widget == nil {
		return 0
	}
	return gtk.BaseWidget(widget).AllocatedHeight()
}

func menuMaxHeight(host gtk.Widgetter) int {
	h := widgetHeight(host)
	if h < 240 {
		h = 240
	}
	if h > 48 {
		return h - 24
	}
	return h
}

func unclipPopoverMenu(pop *gtk.PopoverMenu, maxH int) {
	if pop == nil {
		return
	}
	if maxH < 240 {
		maxH = 240
	}
	visitWidgets(&pop.Widget, func(widget gtk.Widgetter) {
		sw := asScrolledWindow(widget)
		if sw == nil {
			return
		}
		sw.SetPropagateNaturalWidth(true)
		sw.SetPropagateNaturalHeight(true)
		sw.SetMaxContentHeight(maxH)
		sw.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	})
}

func visitWidgets(widget gtk.Widgetter, fn func(gtk.Widgetter)) {
	if widget == nil {
		return
	}
	fn(widget)
	base := gtk.BaseWidget(widget)
	for child := base.FirstChild(); child != nil; child = gtk.BaseWidget(child).NextSibling() {
		visitWidgets(child, fn)
	}
}

func asScrolledWindow(widget gtk.Widgetter) *gtk.ScrolledWindow {
	if widget == nil {
		return nil
	}
	if sw, ok := widget.(*gtk.ScrolledWindow); ok {
		return sw
	}
	object := glib.BaseObject(widget)
	if object == nil {
		return nil
	}
	casted := object.WalkCast(func(obj glib.Objector) bool {
		_, ok := obj.(*gtk.ScrolledWindow)
		return ok
	})
	sw, _ := casted.(*gtk.ScrolledWindow)
	return sw
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
