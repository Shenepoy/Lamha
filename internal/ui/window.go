// Package ui provides Lamha's GTK4 desktop interface.
package ui

import (
	"context"
	"errors"
	"log"
	"path/filepath"
	"time"

	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdkwayland/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/autostart"
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
	editor              *editor
	overlay             *captureOverlay
	busy                bool
	restoreAfterCapture bool
	shortcuts           *gtk.Window
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

	header := gtk.NewHeaderBar()
	header.SetShowTitleButtons(true)
	titleBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	titleBox.Append(brand.Image(28))
	appName := gtk.NewLabel("Lamha")
	appName.SetCSSClasses([]string{"title-3"})
	titleBox.Append(appName)
	header.SetTitleWidget(titleBox)
	shortcutBtn := gtk.NewButtonWithLabel(i18n.T("Shortcuts"))
	shortcutBtn.SetTooltipText(i18n.T("Edit capture and markup keyboard shortcuts"))
	shortcutBtn.ConnectClicked(w.openShortcutSettings)
	header.PackStart(shortcutBtn)
	settingsBtn := gtk.NewButtonWithLabel(i18n.T("Settings"))
	settingsBtn.SetTooltipText(i18n.T("Change magnifier zoom and other options"))
	settingsBtn.ConnectClicked(w.openSettings)
	header.PackStart(settingsBtn)
	quit := gtk.NewButtonWithLabel(i18n.T("Quit"))
	quit.SetTooltipText(i18n.T("Exit Lamha completely"))
	quit.ConnectClicked(func() {
		if app := w.window.Application(); app != nil {
			app.Quit()
		}
	})
	header.PackEnd(quit)
	w.window.SetTitlebar(header)

	root := gtk.NewBox(gtk.OrientationVertical, 18)
	root.SetMarginTop(24)
	root.SetMarginBottom(24)
	root.SetMarginStart(24)
	root.SetMarginEnd(24)

	title := gtk.NewLabel(i18n.T("Capture what matters"))
	alignStart(title)
	title.SetCSSClasses([]string{"title-1"})

	description := gtk.NewLabel(i18n.T("Lamha stays in the background. Close this window to hide it. Capture from the tray, or use the system shortcuts. Open Shortcuts to change every keybind. Quit here or from the tray to exit."))
	alignStart(description)
	description.SetWrap(true)
	description.SetCSSClasses([]string{"dim-label"})

	hero := gtk.NewBox(gtk.OrientationHorizontal, 16)
	hero.Append(brand.Image(88))
	intro := gtk.NewBox(gtk.OrientationVertical, 8)
	intro.SetHExpand(true)
	intro.Append(title)
	intro.Append(description)
	hero.Append(intro)
	root.Append(hero)

	actions := gtk.NewBox(gtk.OrientationHorizontal, 12)
	w.captureArea = gtk.NewButtonWithLabel(i18n.T("Capture area"))
	w.captureArea.SetCSSClasses([]string{"suggested-action"})
	w.captureArea.SetTooltipText(i18n.T("Freeze the screen, then select and mark up an area in Lamha"))
	w.captureArea.ConnectClicked(func() { w.StartCapture(CaptureArea) })
	actions.Append(w.captureArea)

	w.captureWindow = gtk.NewButtonWithLabel(i18n.T("Capture window"))
	w.captureWindow.SetTooltipText(i18n.T("Freeze the screen, then select a window area in Lamha"))
	w.captureWindow.ConnectClicked(func() { w.StartCapture(CaptureWindow) })
	actions.Append(w.captureWindow)

	w.captureScreen = gtk.NewButtonWithLabel(i18n.T("Capture screen"))
	w.captureScreen.SetTooltipText(i18n.T("Freeze the screen, then mark it up in Lamha"))
	w.captureScreen.ConnectClicked(func() { w.StartCapture(CaptureScreen) })
	actions.Append(w.captureScreen)

	w.delay = gtk.NewDropDownFromStrings(delayLabels())
	w.delay.SetSelected(0)
	w.delay.SetTooltipText(i18n.T("Wait before capture so menus and hover states can appear"))
	actions.Append(w.delay)
	root.Append(actions)

	statusRow := gtk.NewBox(gtk.OrientationHorizontal, 8)
	w.spinner = gtk.NewSpinner()
	statusRow.Append(w.spinner)
	w.status = gtk.NewLabel(i18n.T("Ready to capture."))
	alignStart(w.status)
	w.status.SetWrap(true)
	statusRow.Append(w.status)
	root.Append(statusRow)

	paned := gtk.NewPaned(gtk.OrientationHorizontal)
	paned.SetHExpand(true)
	paned.SetVExpand(true)
	paned.SetResizeStartChild(false)

	historyFrame := gtk.NewFrame(i18n.T("History"))
	scroll := gtk.NewScrolledWindow()
	scroll.SetPolicy(gtk.PolicyAutomatic, gtk.PolicyAutomatic)
	scroll.SetMinContentWidth(240)
	scroll.SetSizeRequest(240, 320)
	w.history = gtk.NewListBox()
	w.history.SetSelectionMode(gtk.SelectionSingle)
	w.history.SetShowSeparators(true)
	w.history.ConnectRowSelected(w.historySelected)
	scroll.SetChild(w.history)
	historyFrame.SetChild(scroll)
	paned.SetStartChild(historyFrame)

	previewFrame := gtk.NewFrame(i18n.T("Latest capture"))
	previewFrame.SetHExpand(true)
	previewFrame.SetVExpand(true)
	w.preview = gtk.NewPicture()
	w.preview.SetHExpand(true)
	w.preview.SetVExpand(true)
	w.preview.SetCanShrink(true)
	w.preview.SetContentFit(gtk.ContentFitContain)
	w.preview.SetSizeRequest(480, 320)
	previewFrame.SetChild(w.preview)
	paned.SetEndChild(previewFrame)
	root.Append(paned)

	footer := gtk.NewBox(gtk.OrientationHorizontal, 12)
	w.copyButton = gtk.NewButtonWithLabel(i18n.T("Copy image"))
	w.copyButton.SetSensitive(false)
	w.copyButton.ConnectClicked(w.copyLatest)
	footer.Append(w.copyButton)
	w.annotateBtn = gtk.NewButtonWithLabel(i18n.T("Annotate"))
	w.annotateBtn.SetSensitive(false)
	w.annotateBtn.SetTooltipText(i18n.T("Open the markup tools on this capture"))
	w.annotateBtn.ConnectClicked(func() {
		if w.lastCapture != nil {
			w.openEditor(w.lastCapture.Path)
		}
	})
	footer.Append(w.annotateBtn)
	w.openButton = gtk.NewButtonWithLabel(i18n.T("Open captures folder"))
	w.openButton.ConnectClicked(w.openCaptureFolder)
	footer.Append(w.openButton)

	auto := gtk.NewCheckButtonWithLabel(i18n.T("Start in the background on login"))
	auto.SetActive(autostart.Enabled())
	auto.ConnectToggled(func() {
		if err := autostart.SetEnabled(auto.Active()); err != nil {
			w.status.SetText(i18n.Tf("Could not update login start: %v", err))
		}
	})
	footer.Append(auto)
	root.Append(footer)

	w.window.SetChild(root)
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
	w.copyButton.SetSensitive(true)
	w.annotateBtn.SetSensitive(true)
	if message != "" {
		w.status.SetText(message)
	}
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
		w.history.Append(newHistoryRow(item))
	}

	if len(items) == 0 {
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
		return
	}
	index := row.Index()
	if index < 0 || index >= len(w.historyItems) {
		return
	}
	saved := w.historyItems[index]
	w.showCapture(saved, "")
}

func newHistoryRow(item capture.SavedCapture) *gtk.ListBoxRow {
	row := gtk.NewListBoxRow()
	box := gtk.NewBox(gtk.OrientationVertical, 2)
	box.SetMarginTop(8)
	box.SetMarginBottom(8)
	box.SetMarginStart(10)
	box.SetMarginEnd(10)

	name := gtk.NewLabel(filepath.Base(item.Path))
	alignStart(name)
	name.SetWrap(true)
	box.Append(name)

	when := gtk.NewLabel(item.CreatedAt.Format("2006-01-02 15:04:05"))
	alignStart(when)
	when.SetCSSClasses([]string{"dim-label"})
	box.Append(when)

	row.SetChild(box)
	return row
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
