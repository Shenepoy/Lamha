// Package ui provides Lamha's GTK4 desktop interface.
package ui

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdkwayland/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	glibv2 "github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"

	"github.com/lamha-app/lamha/internal/brand"
	"github.com/lamha-app/lamha/internal/capture"
	"github.com/lamha-app/lamha/internal/grab"
	"github.com/lamha-app/lamha/internal/i18n"
	"github.com/lamha-app/lamha/internal/portal"
)

// Window is Lamha's main application window.
type Window struct {
	window              *gtk.ApplicationWindow
	store               *capture.Store
	preview             *gtk.Picture
	previewStack        *gtk.Stack
	status              *gtk.Label
	spinner             *gtk.Spinner
	delay               *gtk.DropDown
	history             *gtk.ListBox
	historyItems        []capture.SavedCapture
	captureArea         *gtk.Button
	captureWindow       *gtk.Button
	captureScreen       *gtk.Button
	copyButton          *gtk.Button
	annotateBtn         *gtk.Button
	openButton          *gtk.Button
	lastCapture         *capture.SavedCapture
	menuTarget          *capture.SavedCapture
	editor              *editor
	overlay             *captureOverlay
	busy                bool
	restoreAfterCapture bool
	shortcuts           *gtk.Window
	historyMenu         *gtk.PopoverMenu
	exportedHandle      string
	exportedTop         *gdkwayland.WaylandToplevel
}

// New builds Lamha's main window and its local capture store.
func New(application *gtk.Application) (*Window, error) {
	store, err := capture.NewStore("")
	if err != nil {
		return nil, err
	}

	w := &Window{
		window: gtk.NewApplicationWindow(application),
		store:  store,
	}
	applyAppDirection()
	brand.ApplyIconTheme()
	w.build()
	w.refreshHistory("")
	return w, nil
}

// Present makes the main window visible.
func (w *Window) Present() {
	w.window.Present()
}

// Hide sends Lamha to the background indicator.
func (w *Window) Hide() {
	w.window.SetVisible(false)
}

// IsShown reports whether the main window is on screen.
func (w *Window) IsShown() bool {
	return w.window.IsVisible()
}

// Notify shows a desktop notification when the window is hidden.
func (w *Window) Notify(title, body string) {
	note := gio.NewNotification(title)
	note.SetBody(body)
	note.SetIcon(gio.NewThemedIcon(brand.Name))
	if app := w.window.Application(); app != nil {
		app.SendNotification("lamha", note)
	}
}

// StartCapture begins a capture without forcing the main window open.
func (w *Window) StartCapture(mode CaptureMode) {
	if mode == CaptureNone {
		w.Present()
		return
	}
	w.startCapture(mode)
}

func (w *Window) build() {
	w.window.SetTitle("Lamha")
	applyDirection(&w.window.Widget)
	w.window.SetDefaultSize(1040, 680)
	w.window.SetIconName(brand.Name)
	w.window.SetHideOnClose(true)
	w.window.ConnectCloseRequest(func() bool {
		w.Hide()
		return true
	})

	w.installWindowActions()

	header := gtk.NewHeaderBar()
	header.SetShowTitleButtons(true)

	modes := gtk.NewBox(gtk.OrientationHorizontal, 0)
	modes.SetCSSClasses([]string{"linked"})
	modes.SetHAlign(gtk.AlignCenter)
	w.captureArea = gtk.NewButtonWithLabel(i18n.T("Area"))
	w.captureArea.SetTooltipText(i18n.T("Freeze the screen, then select and mark up an area in Lamha"))
	w.captureArea.ConnectClicked(func() { w.StartCapture(CaptureArea) })
	modes.Append(w.captureArea)
	w.captureWindow = gtk.NewButtonWithLabel(i18n.T("Window"))
	w.captureWindow.SetTooltipText(i18n.T("Freeze the screen, then select a window area in Lamha"))
	w.captureWindow.ConnectClicked(func() { w.StartCapture(CaptureWindow) })
	modes.Append(w.captureWindow)
	w.captureScreen = gtk.NewButtonWithLabel(i18n.T("Screen"))
	w.captureScreen.SetTooltipText(i18n.T("Freeze the screen, then mark it up in Lamha"))
	w.captureScreen.ConnectClicked(func() { w.StartCapture(CaptureScreen) })
	modes.Append(w.captureScreen)
	header.SetTitleWidget(modes)

	statusRow := gtk.NewBox(gtk.OrientationHorizontal, 8)
	statusRow.SetVAlign(gtk.AlignCenter)
	w.spinner = gtk.NewSpinner()
	statusRow.Append(w.spinner)
	w.status = gtk.NewLabel(i18n.T("Ready to capture."))
	alignStart(w.status)
	w.status.SetEllipsize(pango.EllipsizeEnd)
	w.status.SetCSSClasses([]string{"dim-label"})
	statusRow.Append(w.status)
	header.PackStart(statusRow)

	menuBtn := gtk.NewMenuButton()
	menuBtn.SetIconName("open-menu-symbolic")
	menuBtn.SetTooltipText(i18n.T("Main menu"))
	menuBtn.SetPrimary(true)
	menuBtn.SetHasFrame(false)
	menuBtn.SetMenuModel(w.primaryMenu())
	header.PackEnd(menuBtn)

	delayBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	delayBox.SetVAlign(gtk.AlignCenter)
	delayLabel := gtk.NewLabel(i18n.T("Delay"))
	delayLabel.SetCSSClasses([]string{"dim-label"})
	delayBox.Append(delayLabel)
	w.delay = gtk.NewDropDownFromStrings(delayLabels())
	w.delay.SetSelected(0)
	w.delay.SetTooltipText(i18n.T("Wait before capture so menus and hover states can appear"))
	delayBox.Append(w.delay)
	header.PackEnd(delayBox)
	w.window.SetTitlebar(header)

	root := gtk.NewBox(gtk.OrientationVertical, 8)
	root.SetMarginTop(12)
	root.SetMarginBottom(12)
	root.SetMarginStart(12)
	root.SetMarginEnd(12)

	paned := gtk.NewPaned(gtk.OrientationHorizontal)
	paned.SetHExpand(true)
	paned.SetVExpand(true)
	paned.SetResizeStartChild(false)
	paned.SetStartChild(w.buildHistoryPane())
	paned.SetEndChild(w.buildPreviewPane())
	root.Append(paned)

	w.window.SetChild(root)
}

func (w *Window) installWindowActions() {
	add := func(name string, fn func()) {
		action := gio.NewSimpleAction(name, nil)
		action.ConnectActivate(func(*glibv2.Variant) { fn() })
		w.window.AddAction(action)
	}
	add("shortcuts", w.openShortcutSettings)
	add("settings", w.openSettings)
	add("open-folder", w.openCaptureFolder)
	add("annotate-capture", w.annotateSelected)
	add("copy-image", w.copyLatest)
	add("copy-capture", w.copyContextImage)
	add("copy-path", w.copySelectedPath)
	add("copy-name", w.copySelectedName)
	add("open-capture", w.openSelectedFile)
	add("delete-capture", w.deleteSelected)
	add("quit", func() {
		if app := w.window.Application(); app != nil {
			app.Quit()
		}
	})
}

func (w *Window) primaryMenu() gio.MenuModeller {
	app := gio.NewMenu()
	app.Append(i18n.T("Keyboard shortcuts"), "win.shortcuts")
	app.Append(i18n.T("Settings"), "win.settings")
	app.Append(i18n.T("Open captures folder"), "win.open-folder")

	quit := gio.NewMenu()
	quit.Append(i18n.T("Quit"), "win.quit")

	menu := gio.NewMenu()
	menu.AppendSection("", app)
	menu.AppendSection("", quit)
	return menu
}

func (w *Window) buildHistoryPane() gtk.Widgetter {
	column := gtk.NewBox(gtk.OrientationVertical, 8)

	title := gtk.NewLabel(i18n.T("Recent"))
	alignStart(title)
	title.SetCSSClasses([]string{"title-4"})
	column.Append(title)

	scroll := gtk.NewScrolledWindow()
	scroll.SetPolicy(gtk.PolicyAutomatic, gtk.PolicyAutomatic)
	scroll.SetMinContentWidth(280)
	scroll.SetSizeRequest(280, 320)
	scroll.SetHasFrame(true)
	scroll.SetHExpand(true)
	scroll.SetVExpand(true)

	w.history = gtk.NewListBox()
	w.history.SetSelectionMode(gtk.SelectionSingle)
	w.history.SetShowSeparators(false)
	w.history.SetCSSClasses([]string{"navigation-sidebar"})
	placeholder := gtk.NewLabel(i18n.T("No captures yet"))
	placeholder.SetCSSClasses([]string{"dim-label"})
	placeholder.SetWrap(true)
	placeholder.SetJustify(gtk.JustifyCenter)
	w.history.SetPlaceholder(&placeholder.Widget)
	w.history.ConnectRowSelected(w.historySelected)
	scroll.SetChild(w.history)
	column.Append(scroll)
	column.SetMarginEnd(16)
	return column
}

func (w *Window) buildPreviewPane() gtk.Widgetter {
	column := gtk.NewBox(gtk.OrientationVertical, 0)
	column.SetHExpand(true)
	column.SetVExpand(true)
	column.SetMarginStart(16)

	w.preview = gtk.NewPicture()
	w.preview.SetHExpand(true)
	w.preview.SetVExpand(true)
	w.preview.SetCanShrink(true)
	w.preview.SetContentFit(gtk.ContentFitContain)
	w.preview.SetSizeRequest(480, 320)

	empty := gtk.NewLabel(i18n.T("Select a capture to preview it."))
	empty.SetCSSClasses([]string{"dim-label"})
	empty.SetWrap(true)
	empty.SetHAlign(gtk.AlignCenter)
	empty.SetVAlign(gtk.AlignCenter)
	empty.SetJustify(gtk.JustifyCenter)

	w.previewStack = gtk.NewStack()
	w.previewStack.SetHExpand(true)
	w.previewStack.SetVExpand(true)
	w.previewStack.AddNamed(empty, "empty")
	w.previewStack.AddNamed(w.preview, "image")
	w.previewStack.SetVisibleChildName("empty")
	w.attachPreviewMenu()
	column.Append(w.previewStack)

	bar := gtk.NewActionBar()
	w.copyButton = gtk.NewButtonWithLabel(i18n.T("Copy image"))
	w.copyButton.SetCSSClasses([]string{"suggested-action"})
	w.copyButton.SetSensitive(false)
	w.copyButton.ConnectClicked(w.copyLatest)
	bar.PackStart(w.copyButton)
	w.annotateBtn = gtk.NewButtonWithLabel(i18n.T("Annotate"))
	w.annotateBtn.SetSensitive(false)
	w.annotateBtn.SetTooltipText(i18n.T("Open the markup tools on this capture"))
	w.annotateBtn.ConnectClicked(func() {
		if w.lastCapture != nil {
			w.openEditor(w.lastCapture.Path)
		}
	})
	bar.PackStart(w.annotateBtn)
	w.openButton = gtk.NewButtonWithLabel(i18n.T("Open captures folder"))
	w.openButton.ConnectClicked(w.openCaptureFolder)
	bar.PackEnd(w.openButton)
	column.Append(bar)
	return column
}

func (w *Window) startCapture(mode CaptureMode) {
	if w.busy {
		return
	}

	delay := delayFromIndex(w.delay.Selected())
	w.restoreAfterCapture = w.window.IsVisible()
	w.setBusy(true, i18n.T("Capturing screen…"))
	if w.restoreAfterCapture {
		w.window.Present()
	}
	log.Printf("starting capture mode=%d", mode)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				w.captureFailed(ctx.Err())
				return
			}
			timer.Stop()
		}

		log.Printf("trying silent compositor grab")
		staging, err := grab.Fast(ctx)
		if err != nil {
			log.Printf("silent grab failed: %v", err)
			glib.IdleAdd(func() {
				w.status.SetText(i18n.T("Requesting screenshot access…"))
			})
			parent, drop := w.portalParentIfMapped(ctx)
			defer drop()
			log.Printf("portal parent=%q", parent)

			staging, err = grab.ViaPortal(ctx, portal.ScreenshotOptions{ParentWindow: parent})
			if err != nil && !errors.Is(err, portal.ErrCancelled) {
				log.Printf("silent portal failed; asking GNOME for permission: %v", err)
				glib.IdleAdd(func() {
					w.status.SetText(i18n.T("GNOME needs one-time permission. Allow the system screenshot dialog."))
				})
				staging, err = grab.ViaPortal(ctx, portal.ScreenshotOptions{
					ParentWindow: parent,
					Interactive:  true,
				})
			}
			if err != nil {
				log.Printf("portal grab failed: %v", err)
			}
		}

		glib.IdleAdd(func() {
			w.setBusy(false, "")
			if err != nil {
				w.failCapture(err)
				return
			}
			log.Printf("opening capture overlay")
			w.openCaptureOverlay(mode, staging)
		})
	}()
}

func (w *Window) captureFailed(err error) {
	glib.IdleAdd(func() {
		w.setBusy(false, "")
		w.failCapture(err)
	})
}

func (w *Window) setBusy(busy bool, message string) {
	w.busy = busy
	w.captureArea.SetSensitive(!busy)
	w.captureWindow.SetSensitive(!busy)
	w.captureScreen.SetSensitive(!busy)
	w.delay.SetSensitive(!busy)
	if busy {
		w.spinner.Start()
		if message != "" {
			w.status.SetText(message)
		}
		return
	}
	w.spinner.Stop()
}

func (w *Window) failCapture(err error) {
	message := i18n.Tf("Could not capture: %v", err)
	switch {
	case errors.Is(err, portal.ErrCancelled):
		message = i18n.T("Capture cancelled.")
	case errors.Is(err, portal.ErrDenied):
		message = i18n.T("GNOME denied the screenshot. Allow Lamha in the system dialog, or Settings → Privacy → Screen, then try again.")
	case errors.Is(err, context.DeadlineExceeded):
		message = i18n.T("Capture timed out. Please try again.")
	}
	w.status.SetText(message)
	if w.restoreAfterCapture {
		w.Present()
		return
	}
	w.Notify("Lamha", message)
}

func (w *Window) showCapture(saved capture.SavedCapture, message string) {
	w.lastCapture = &saved
	w.preview.SetFilename(saved.Path)
	w.previewStack.SetVisibleChildName("image")
	w.copyButton.SetSensitive(true)
	w.annotateBtn.SetSensitive(true)
	if message != "" {
		w.status.SetText(message)
	}
}

func (w *Window) showEmptyPreview() {
	w.lastCapture = nil
	w.preview.SetFilename("")
	w.previewStack.SetVisibleChildName("empty")
	w.copyButton.SetSensitive(false)
	w.annotateBtn.SetSensitive(false)
}

func (w *Window) afterAnnotation(path string, copied bool) {
	saved := capture.SavedCapture{Path: path, URI: fileURI(path)}
	if w.lastCapture != nil && w.lastCapture.Path == path {
		saved = *w.lastCapture
	}
	message := i18n.Tf("Saved %s", path)
	if copied {
		message = i18n.Tf("Saved %s and copied it to the clipboard.", path)
	}
	w.showCapture(saved, message)
	w.refreshHistory(path)
}

func (w *Window) refreshHistory(selectPath string) {
	items, err := w.store.List()
	if err != nil {
		w.status.SetText(i18n.Tf("Could not load capture history: %v", err))
		return
	}

	w.historyItems = items
	w.history.RemoveAll()
	for _, item := range items {
		w.history.Append(w.newHistoryRow(item))
	}

	if len(items) == 0 {
		w.showEmptyPreview()
		return
	}

	index := 0
	if selectPath != "" {
		for i, item := range items {
			if item.Path == selectPath {
				index = i
				break
			}
		}
	}
	if row := w.history.RowAtIndex(index); row != nil {
		w.history.SelectRow(row)
	}
}

func (w *Window) historySelected(row *gtk.ListBoxRow) {
	if row == nil {
		w.showEmptyPreview()
		return
	}
	index := row.Index()
	if index < 0 || index >= len(w.historyItems) {
		return
	}
	saved := w.historyItems[index]
	w.showCapture(saved, "")
}

func (w *Window) copyLatest() {
	if w.lastCapture == nil {
		return
	}
	if err := copyImageFile(w.lastCapture.Path); err != nil {
		w.status.SetText(i18n.Tf("Could not copy image: %v", err))
		return
	}
	w.status.SetText(i18n.T("Image copied to clipboard."))
}

func (w *Window) openCaptureFolder() {
	if err := gio.AppInfoLaunchDefaultForURI(fileURI(w.store.Directory()), nil); err != nil {
		w.status.SetText(i18n.Tf("Could not open captures folder: %v", err))
	}
}

func fileURI(path string) string {
	// gio.File produces an RFC-compliant URI and handles spaces correctly.
	return gio.NewFileForPath(path).URI()
}
