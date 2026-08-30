package ui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"

	"github.com/lamha-app/lamha/internal/autostart"
	"github.com/lamha-app/lamha/internal/brand"
	"github.com/lamha-app/lamha/internal/capture"
	"github.com/lamha-app/lamha/internal/i18n"
	"github.com/lamha-app/lamha/internal/prefs"
	"github.com/lamha-app/lamha/internal/version"
)

var prefsCSSOnce sync.Once

func ensurePrefsCSS() {
	prefsCSSOnce.Do(func() {
		provider := gtk.NewCSSProvider()
		provider.LoadFromData(`
.lamha-prefs {
  padding: 8px 20px 28px;
}
.lamha-prefs-group {
  margin-top: 18px;
}
.lamha-prefs > .lamha-prefs-group:first-child {
  margin-top: 8px;
}
.lamha-prefs-heading {
  margin: 0 12px 8px;
}
.lamha-prefs-lead {
  margin: 0 12px 10px;
}
.lamha-prefs-card {
  background-color: @theme_base_color;
  color: @theme_text_color;
  border: 1px solid @borders;
  border-radius: 12px;
}
.lamha-prefs-row {
  min-height: 52px;
  padding: 10px 16px;
}
.lamha-prefs-title {
  font-weight: 500;
}
.lamha-prefs-subtitle {
  font-size: 0.9em;
}
.lamha-prefs-value {
  font-features: tnum;
  font-weight: 600;
  min-width: 3.2em;
}
.lamha-prefs-card > .lamha-prefs-row + .lamha-prefs-row,
.lamha-prefs-card > .lamha-prefs-row + button,
.lamha-prefs-card > button + .lamha-prefs-row,
.lamha-prefs-card > button + button {
  border-top: 1px solid @borders;
}
.lamha-prefs-card > button.lamha-prefs-action {
  padding: 0;
  border-radius: 0;
}
.lamha-prefs-card > button.lamha-prefs-action:first-child {
  border-top-left-radius: 12px;
  border-top-right-radius: 12px;
}
.lamha-prefs-card > button.lamha-prefs-action:last-child {
  border-bottom-left-radius: 12px;
  border-bottom-right-radius: 12px;
}
.lamha-prefs-banner {
  background-color: @theme_base_color;
  color: @theme_text_color;
  border: 1px solid @borders;
  border-radius: 12px;
  padding: 12px 14px;
  margin: 16px 0 4px;
}
.lamha-prefs-control {
  min-width: 148px;
}
.lamha-key {
  min-width: 148px;
  font-weight: 600;
}
@media (prefers-contrast: more) {
  .lamha-prefs-card,
  .lamha-prefs-banner {
    border-width: 2px;
  }
}
`)
		if display := gdk.DisplayGetDefault(); display != nil {
			gtk.StyleContextAddProviderForDisplay(display, provider, gtk.STYLE_PROVIDER_PRIORITY_USER)
		}
	})
}

func (w *Window) openSettings() {
	if w.settingsWin != nil {
		w.settingsWin.Present()
		return
	}

	win := gtk.NewWindow()
	if app := w.window.Application(); app != nil {
		win.SetApplication(app)
	}
	win.SetTitle(i18n.T("Settings"))
	win.SetIconName(brand.Name)
	applyDirection(&win.Widget)
	win.SetTransientFor(&w.window.Window)
	win.SetModal(true)
	win.SetDefaultSize(480, 720)
	win.SetHideOnClose(true)
	win.ConnectCloseRequest(func() bool {
		w.settingsWin = nil
		win.Destroy()
		return true
	})

	header := gtk.NewHeaderBar()
	win.SetTitlebar(header)
	win.SetChild(w.settingsRoot())
	w.settingsWin = win
	win.Present()
}

func (w *Window) rebuildSettings() {
	if w.settingsWin == nil {
		return
	}
	w.settingsWin.SetTitle(i18n.T("Settings"))
	applyDirection(&w.settingsWin.Widget)
	w.settingsWin.SetChild(w.settingsRoot())
}

func (w *Window) settingsRoot() *gtk.ScrolledWindow {
	ensurePrefsCSS()
	scroll := gtk.NewScrolledWindow()
	scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scroll.SetChild(w.buildSettingsBody())
	return scroll
}

func (w *Window) buildSettingsBody() *gtk.Box {
	page := prefsPage()

	look, lookCard := prefsGroup(i18n.T("Appearance"), "")
	theme := prefsDropDown([]string{i18n.T("System"), i18n.T("Light"), i18n.T("Dark")}, themeIndex(prefs.Current().Theme()))
	theme.Connect("notify::selected", func() {
		value := themeValue(theme.Selected())
		if value == prefs.Current().Theme() {
			return
		}
		if err := prefs.Current().SetTheme(value); err != nil {
			w.status.SetText(i18n.Tf("Could not save settings: %v", err))
			return
		}
		w.setTheme(value)
	})
	lookCard.Append(prefsRow(i18n.T("Theme"), i18n.T("Color scheme for Lamha windows."), theme))

	lang := prefsDropDown([]string{i18n.T("System"), i18n.T("English"), i18n.T("Arabic")}, languageIndex(prefs.Current().Language()))
	lang.Connect("notify::selected", func() {
		value := languageValue(lang.Selected())
		if value == prefs.Current().Language() {
			return
		}
		if err := prefs.Current().SetLanguage(value); err != nil {
			w.status.SetText(i18n.Tf("Could not save settings: %v", err))
			return
		}
		i18n.SetLanguage(value)
		w.refreshLocale()
		w.rebuildSettings()
	})
	lookCard.Append(prefsRow(i18n.T("Language"), i18n.T("Language for buttons, menus, and messages."), lang))
	page.Append(look)

	toolbar, toolbarCard := prefsGroup(i18n.T("Toolbar"), i18n.T("Drag the toolbar anywhere. It snaps to an edge when you get close."))
	edge := prefsDropDown([]string{
		i18n.T("Top"),
		i18n.T("Bottom"),
		i18n.T("Left"),
		i18n.T("Right"),
		i18n.T("Free"),
	}, toolbarEdgeIndex(prefs.Current().ToolbarEdge()))

	alignHelp := gtk.NewLabel(toolbarAlignHelp(prefs.Current().ToolbarEdge()))
	align := prefsDropDown(toolbarAlignLabels(prefs.Current().ToolbarEdge()), toolbarAlignIndex(prefs.Current().ToolbarOffset()))
	alignRow := prefsRowWithHelp(i18n.T("Along the edge"), alignHelp, align)
	syncingAlign := false
	setAlignVisible := func(edgeName string) {
		alignRow.SetVisible(!prefs.ToolbarFloating(edgeName))
	}
	setAlignVisible(prefs.Current().ToolbarEdge())

	edge.Connect("notify::selected", func() {
		value := toolbarEdgeValue(edge.Selected())
		if value == prefs.Current().ToolbarEdge() {
			return
		}
		if err := prefs.Current().SetToolbarPlacement(value, prefs.Current().ToolbarX(), prefs.Current().ToolbarY()); err != nil {
			w.status.SetText(i18n.Tf("Could not save settings: %v", err))
			return
		}
		syncingAlign = true
		align.SetModel(gtk.NewStringList(toolbarAlignLabels(value)))
		align.SetSelected(toolbarAlignIndex(prefs.Current().ToolbarOffset()))
		syncingAlign = false
		alignHelp.SetText(toolbarAlignHelp(value))
		setAlignVisible(value)
	})
	align.Connect("notify::selected", func() {
		if syncingAlign {
			return
		}
		next := toolbarAlignValue(align.Selected())
		if next == prefs.ToolbarAlignForOffset(prefs.Current().ToolbarOffset()) {
			return
		}
		x, y := prefs.Current().ToolbarX(), prefs.Current().ToolbarY()
		offset := prefs.ToolbarOffsetForAlign(next)
		if prefs.ToolbarVertical(prefs.Current().ToolbarEdge()) {
			y = offset
		} else {
			x = offset
		}
		if err := prefs.Current().SetToolbarPlacement(prefs.Current().ToolbarEdge(), x, y); err != nil {
			w.status.SetText(i18n.Tf("Could not save settings: %v", err))
		}
	})
	toolbarCard.Append(prefsRow(i18n.T("Screen edge"), i18n.T("Top and bottom keep the toolbar horizontal. Left and right turn it vertical. Free lets it sit anywhere."), edge))
	toolbarCard.Append(alignRow)

	snap := prefsSwitch(prefs.Current().ToolbarSnapAlign())
	snap.Connect("notify::active", func() {
		if snap.Active() == prefs.Current().ToolbarSnapAlign() {
			return
		}
		if err := prefs.Current().SetToolbarSnapAlign(snap.Active()); err != nil {
			w.status.SetText(i18n.Tf("Could not save settings: %v", err))
		}
	})
	toolbarCard.Append(prefsRow(i18n.T("Snap when dropped"), i18n.T("Snap to the middle and corners when dropped"), snap))

	lock := prefsSwitch(prefs.Current().ToolbarLocked())
	lock.Connect("notify::active", func() {
		if lock.Active() == prefs.Current().ToolbarLocked() {
			return
		}
		if err := prefs.Current().SetToolbarLocked(lock.Active()); err != nil {
			w.status.SetText(i18n.Tf("Could not save settings: %v", err))
		}
	})
	toolbarCard.Append(prefsRow(i18n.T("Lock toolbar in place"), i18n.T("Keep the toolbar where you left it."), lock))

	reset := gtk.NewButtonWithLabel(i18n.T("Reset"))
	reset.SetVAlign(gtk.AlignCenter)
	reset.SetTooltipText(i18n.T("Move the toolbar back to the top center."))
	reset.ConnectClicked(func() {
		if err := prefs.Current().ResetToolbar(); err != nil {
			w.status.SetText(i18n.Tf("Could not save settings: %v", err))
			return
		}
		w.rebuildSettings()
	})
	toolbarCard.Append(prefsRow(i18n.T("Reset toolbar position"), i18n.T("Move the toolbar back to the top center."), reset))
	page.Append(toolbar)

	capture, captureCard := prefsGroup(i18n.T("Capture"), "")
	zoomValue := gtk.NewLabel(zoomCaption(prefs.Current().MagnifierZoom()))
	styleDim(zoomValue)
	zoomValue.AddCSSClass("numeric")
	zoomValue.AddCSSClass("lamha-prefs-value")
	zoomValue.SetXAlign(1)
	zoomValue.SetVAlign(gtk.AlignCenter)

	scale := gtk.NewScaleWithRange(gtk.OrientationHorizontal, prefs.MinMagnifierZoom, prefs.MaxMagnifierZoom, 0.5)
	scale.SetDrawValue(false)
	scale.SetHExpand(true)
	scale.SetValue(prefs.Current().MagnifierZoom())
	scale.AddMark(prefs.DefaultMagnifierZoom, gtk.PosBottom, i18n.T("Default"))
	ignoreScaleWheel(scale)
	scale.ConnectValueChanged(func() {
		zoom := prefs.ClampZoom(scale.Value())
		if err := prefs.Current().SetMagnifierZoom(zoom); err != nil {
			w.status.SetText(i18n.Tf("Could not save settings: %v", err))
			return
		}
		zoomValue.SetText(zoomCaption(zoom))
	})

	zoomCol := gtk.NewBox(gtk.OrientationVertical, 4)
	zoomCol.AddCSSClass("lamha-prefs-row")
	zoomHead := prefsText(i18n.T("Magnifier zoom"), i18n.T("How much the capture lens enlarges pixels under the pointer."))
	zoomTop := gtk.NewBox(gtk.OrientationHorizontal, 12)
	zoomTop.SetVAlign(gtk.AlignCenter)
	zoomTop.Append(zoomHead)
	zoomTop.Append(zoomValue)
	zoomCol.Append(zoomTop)
	zoomCol.Append(scale)
	captureCard.Append(zoomCol)

	savePath := prefs.Current().SaveDirectory()
	resetDir := gtk.NewButton()
	refresh := gtk.NewImageFromIconName("view-refresh-symbolic")
	refresh.SetPixelSize(18)
	resetDir.SetChild(refresh)
	resetDir.SetVAlign(gtk.AlignCenter)
	resetDir.SetSensitive(!prefs.Current().SaveDirectoryIsDefault())
	resetDir.SetTooltipText(i18n.T("Use Pictures/Screenshots again."))
	resetDir.ConnectClicked(func() {
		if err := prefs.Current().SetSaveDirectory(""); err != nil {
			w.status.SetText(i18n.Tf("Could not save settings: %v", err))
			return
		}
		if err := w.useSaveDirectory(prefs.Current().SaveDirectory()); err != nil {
			w.status.SetText(i18n.Tf("Could not use that folder: %v", err))
			return
		}
		w.rebuildSettings()
	})
	choose := gtk.NewButtonWithLabel(i18n.T("Choose"))
	choose.SetVAlign(gtk.AlignCenter)
	choose.SetTooltipText(i18n.T("Pick a folder for new screenshots"))
	choose.ConnectClicked(w.chooseSaveDirectory)
	saveActions := gtk.NewBox(gtk.OrientationHorizontal, 8)
	saveActions.SetVAlign(gtk.AlignCenter)
	saveActions.Append(resetDir)
	saveActions.Append(choose)
	captureCard.Append(prefsRow(i18n.T("Save location"), displaySaveDirectory(savePath), saveActions))

	keysRow := prefsActionRow(i18n.T("Keyboard shortcuts"), i18n.T("Edit capture and markup keyboard shortcuts"), w.openShortcutSettings)
	captureCard.Append(keysRow)
	page.Append(capture)

	steps, stepsCard := prefsGroup(i18n.T("Numbered steps"), i18n.T("Click a step with Select or Move to change its number."))
	renumber := prefsSwitch(prefs.Current().RenumberSteps())
	renumber.Connect("notify::active", func() {
		if err := prefs.Current().SetRenumberSteps(renumber.Active()); err != nil {
			w.status.SetText(i18n.Tf("Could not save settings: %v", err))
		}
	})
	stepsCard.Append(prefsRow(i18n.T("Renumber later steps"), i18n.T("When you change one number, every other step shifts by the same amount."), renumber))
	deleteZero := prefsSwitch(prefs.Current().DeleteZeroSteps())
	deleteZero.Connect("notify::active", func() {
		if err := prefs.Current().SetDeleteZeroSteps(deleteZero.Active()); err != nil {
			w.status.SetText(i18n.Tf("Could not save settings: %v", err))
		}
	})
	stepsCard.Append(prefsRow(i18n.T("Delete steps at 0"), i18n.T("Remove a step if you turn its number down to 0."), deleteZero))
	page.Append(steps)

	startup, startupCard := prefsGroup(i18n.T("Startup"), "")
	auto := prefsSwitch(autostart.Enabled())
	auto.Connect("notify::active", func() {
		if err := autostart.SetEnabled(auto.Active()); err != nil {
			w.status.SetText(i18n.Tf("Could not update login start: %v", err))
		}
	})
	startupCard.Append(prefsRow(i18n.T("Start on login"), i18n.T("Start in the background on login"), auto))
	page.Append(startup)

	about, aboutCard := prefsGroup(i18n.T("About"), "")
	aboutCard.Append(prefsActionRow(i18n.T("About Me"), i18n.T("Developer info from GitHub"), w.openAbout))
	aboutCard.Append(prefsActionRow(i18n.T("Source Code"), i18n.T("Browse the app source on GitHub"), func() {
		_ = gio.AppInfoLaunchDefaultForURI(brand.SourceURL, nil)
	}))
	aboutCard.Append(prefsActionRow(i18n.T("Updates"), i18n.T("Check for updates on GitHub"), func() {
		_ = gio.AppInfoLaunchDefaultForURI(brand.UpdateURL, nil)
	}))
	page.Append(about)

	ver := gtk.NewLabel("Lamha " + version.String())
	ver.SetHAlign(gtk.AlignCenter)
	ver.SetMarginTop(18)
	ver.SetCSSClasses([]string{"dim-label", "dimmed", "caption"})
	page.Append(ver)
	return page
}

func ignoreScaleWheel(scale *gtk.Scale) {
	if scale == nil {
		return
	}
	scroll := gtk.NewEventControllerScroll(gtk.EventControllerScrollBothAxes)
	scroll.SetPropagationPhase(gtk.PhaseCapture)
	scroll.ConnectScroll(func(_, _ float64) bool {
		return true
	})
	scale.AddController(scroll)
}

func prefsPage() *gtk.Box {
	page := gtk.NewBox(gtk.OrientationVertical, 0)
	page.AddCSSClass("lamha-prefs")
	return page
}

func prefsGroup(title, lead string) (*gtk.Box, *gtk.Box) {
	wrap := gtk.NewBox(gtk.OrientationVertical, 0)
	wrap.AddCSSClass("lamha-prefs-group")

	heading := gtk.NewLabel(title)
	alignStart(heading)
	heading.AddCSSClass("title-4")
	heading.AddCSSClass("heading")
	heading.AddCSSClass("lamha-prefs-heading")
	wrap.Append(heading)

	if lead != "" {
		help := gtk.NewLabel(lead)
		alignStart(help)
		help.SetWrap(true)
		styleDim(help)
		help.AddCSSClass("lamha-prefs-lead")
		wrap.Append(help)
	}

	card := gtk.NewBox(gtk.OrientationVertical, 0)
	card.AddCSSClass("card")
	card.AddCSSClass("lamha-prefs-card")
	wrap.Append(card)
	return wrap, card
}

func prefsText(title, subtitle string) *gtk.Box {
	var help *gtk.Label
	if subtitle != "" {
		help = gtk.NewLabel(subtitle)
	}
	return prefsTextWithHelp(title, help)
}

func prefsTextWithHelp(title string, help *gtk.Label) *gtk.Box {
	text := gtk.NewBox(gtk.OrientationVertical, 2)
	text.SetHExpand(true)
	text.SetVAlign(gtk.AlignCenter)

	name := gtk.NewLabel(title)
	alignStart(name)
	name.AddCSSClass("heading")
	name.AddCSSClass("lamha-prefs-title")
	name.SetEllipsize(pango.EllipsizeEnd)
	text.Append(name)

	if help != nil {
		alignStart(help)
		help.SetWrap(true)
		styleDim(help)
		help.AddCSSClass("caption")
		help.AddCSSClass("lamha-prefs-subtitle")
		text.Append(help)
	}
	return text
}

func prefsRow(title, subtitle string, suffix gtk.Widgetter) *gtk.Box {
	return prefsRowWithHelp(title, nilIfEmpty(subtitle), suffix)
}

func nilIfEmpty(subtitle string) *gtk.Label {
	if subtitle == "" {
		return nil
	}
	return gtk.NewLabel(subtitle)
}

func prefsRowWithHelp(title string, help *gtk.Label, suffix gtk.Widgetter) *gtk.Box {
	row := gtk.NewBox(gtk.OrientationHorizontal, 16)
	row.AddCSSClass("lamha-prefs-row")
	row.SetVAlign(gtk.AlignCenter)
	row.Append(prefsTextWithHelp(title, help))
	if suffix != nil {
		row.Append(suffix)
	}
	return row
}

func prefsActionRow(title, subtitle string, onClick func()) *gtk.Button {
	btn := gtk.NewButton()
	btn.SetHasFrame(false)
	btn.AddCSSClass("lamha-prefs-action")
	next := gtk.NewImageFromIconName("go-next-symbolic")
	next.SetPixelSize(16)
	styleDim(next)
	next.SetVAlign(gtk.AlignCenter)
	btn.SetChild(prefsRow(title, subtitle, next))
	if onClick != nil {
		btn.ConnectClicked(onClick)
	}
	return btn
}

func prefsDropDown(labels []string, selected uint) *gtk.DropDown {
	drop := gtk.NewDropDownFromStrings(labels)
	drop.SetSelected(selected)
	drop.SetVAlign(gtk.AlignCenter)
	drop.SetHAlign(gtk.AlignEnd)
	drop.AddCSSClass("lamha-prefs-control")
	return drop
}

func prefsSwitch(active bool) *gtk.Switch {
	sw := gtk.NewSwitch()
	sw.SetActive(active)
	sw.SetVAlign(gtk.AlignCenter)
	sw.SetHAlign(gtk.AlignEnd)
	return sw
}

func prefsBanner(text string) *gtk.Label {
	banner := gtk.NewLabel(text)
	banner.SetWrap(true)
	alignStart(banner)
	styleDim(banner)
	banner.AddCSSClass("lamha-prefs-banner")
	return banner
}

func zoomCaption(zoom float64) string {
	return i18n.Tf("%.1f×", prefs.ClampZoom(zoom))
}

func themeIndex(theme string) uint {
	switch prefs.NormalizeTheme(theme) {
	case prefs.ThemeLight:
		return 1
	case prefs.ThemeDark:
		return 2
	default:
		return 0
	}
}

func themeValue(index uint) string {
	switch index {
	case 1:
		return prefs.ThemeLight
	case 2:
		return prefs.ThemeDark
	default:
		return prefs.ThemeSystem
	}
}

func languageIndex(language string) uint {
	switch prefs.NormalizeLanguage(language) {
	case prefs.LanguageEnglish:
		return 1
	case prefs.LanguageArabic:
		return 2
	default:
		return 0
	}
}

func languageValue(index uint) string {
	switch index {
	case 1:
		return prefs.LanguageEnglish
	case 2:
		return prefs.LanguageArabic
	default:
		return prefs.LanguageSystem
	}
}

func toolbarEdgeIndex(edge string) uint {
	switch prefs.NormalizeToolbarEdge(edge) {
	case prefs.ToolbarEdgeBottom:
		return 1
	case prefs.ToolbarEdgeLeft:
		return 2
	case prefs.ToolbarEdgeRight:
		return 3
	case prefs.ToolbarEdgeFloat:
		return 4
	default:
		return 0
	}
}

func toolbarEdgeValue(index uint) string {
	switch index {
	case 1:
		return prefs.ToolbarEdgeBottom
	case 2:
		return prefs.ToolbarEdgeLeft
	case 3:
		return prefs.ToolbarEdgeRight
	case 4:
		return prefs.ToolbarEdgeFloat
	default:
		return prefs.ToolbarEdgeTop
	}
}

func toolbarAlignIndex(offset float64) uint {
	switch prefs.ToolbarAlignForOffset(offset) {
	case prefs.ToolbarAlignEnd:
		return 2
	case prefs.ToolbarAlignStart:
		return 0
	default:
		return 1
	}
}

func toolbarAlignValue(index uint) string {
	switch index {
	case 0:
		return prefs.ToolbarAlignStart
	case 2:
		return prefs.ToolbarAlignEnd
	default:
		return prefs.ToolbarAlignCenter
	}
}

func toolbarAlignLabels(edge string) []string {
	if prefs.ToolbarVertical(edge) {
		return []string{i18n.T("Top"), i18n.T("Center"), i18n.T("Bottom")}
	}
	return []string{i18n.T("Left"), i18n.T("Center"), i18n.T("Right")}
}

func toolbarAlignHelp(edge string) string {
	if prefs.ToolbarVertical(edge) {
		return i18n.T("Top, center, or bottom of that edge.")
	}
	return i18n.T("Left, center, or right of that edge.")
}

func (w *Window) chooseSaveDirectory() {
	dialog := gtk.NewFileDialog()
	dialog.SetTitle(i18n.T("Save screenshots to"))
	dialog.SetAcceptLabel(i18n.T("Choose"))
	dialog.SetInitialFolder(gio.NewFileForPath(prefs.Current().SaveDirectory()))
	parent := &w.window.Window
	if w.settingsWin != nil {
		parent = w.settingsWin
	}
	dialog.SelectFolder(context.Background(), parent, func(res gio.AsyncResulter) {
		folder, err := dialog.SelectFolderFinish(res)
		if err != nil || folder == nil {
			return
		}
		path := folder.Path()
		if path == "" {
			return
		}
		if err := prefs.Current().SetSaveDirectory(path); err != nil {
			w.status.SetText(i18n.Tf("Could not save settings: %v", err))
			return
		}
		if err := w.useSaveDirectory(prefs.Current().SaveDirectory()); err != nil {
			w.status.SetText(i18n.Tf("Could not use that folder: %v", err))
			return
		}
		w.rebuildSettings()
	})
}

func (w *Window) useSaveDirectory(dir string) error {
	store, err := capture.NewStore(dir)
	if err != nil {
		return err
	}
	w.store = store
	w.refreshHistory("")
	return nil
}

func displaySaveDirectory(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	prefix := home + string(os.PathSeparator)
	if strings.HasPrefix(path, prefix) {
		return "~" + string(os.PathSeparator) + filepath.ToSlash(path[len(prefix):])
	}
	return path
}
