package ui

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/autostart"
	"github.com/lamha-app/lamha/internal/brand"
	"github.com/lamha-app/lamha/internal/i18n"
	"github.com/lamha-app/lamha/internal/prefs"
	"github.com/lamha-app/lamha/internal/version"
)

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
	win.SetDefaultSize(440, 520)
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
	scroll := gtk.NewScrolledWindow()
	scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scroll.SetChild(w.buildSettingsBody())
	return scroll
}

func (w *Window) buildSettingsBody() *gtk.Box {
	lookTitle := settingsHeading(i18n.T("Appearance"), 0)
	themeLabel := gtk.NewLabel(i18n.T("Theme"))
	alignStart(themeLabel)
	themeHelp := settingsHelp(i18n.T("Color scheme for Lamha windows."))
	theme := gtk.NewDropDownFromStrings([]string{
		i18n.T("System"),
		i18n.T("Light"),
		i18n.T("Dark"),
	})
	theme.SetSelected(themeIndex(prefs.Current().Theme()))
	theme.Connect("notify::selected", func() {
		value := themeValue(theme.Selected())
		if value == prefs.Current().Theme() {
			return
		}
		if err := prefs.Current().SetTheme(value); err != nil {
			w.status.SetText(i18n.Tf("Could not save settings: %v", err))
			return
		}
		applyTheme(value)
	})

	langLabel := gtk.NewLabel(i18n.T("Language"))
	alignStart(langLabel)
	langLabel.SetMarginTop(8)
	langHelp := settingsHelp(i18n.T("Language for buttons, menus, and messages."))
	lang := gtk.NewDropDownFromStrings([]string{
		i18n.T("System"),
		i18n.T("English"),
		i18n.T("Arabic"),
	})
	lang.SetSelected(languageIndex(prefs.Current().Language()))
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

	zoomTitle := settingsHeading(i18n.T("Capture"), 16)
	zoomLabel := gtk.NewLabel(zoomCaption(prefs.Current().MagnifierZoom()))
	alignStart(zoomLabel)

	help := settingsHelp(i18n.T("How much the capture lens enlarges pixels under the pointer."))

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

	startup := settingsHeading(i18n.T("Startup"), 16)
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
	box.Append(lookTitle)
	box.Append(themeLabel)
	box.Append(themeHelp)
	box.Append(theme)
	box.Append(langLabel)
	box.Append(langHelp)
	box.Append(lang)
	box.Append(zoomTitle)
	box.Append(zoomLabel)
	box.Append(help)
	box.Append(scale)
	box.Append(startup)
	box.Append(auto)

	aboutTitle := settingsHeading(i18n.T("About"), 16)
	aboutHelp := settingsHelp(i18n.T("Developer info from GitHub"))
	aboutBtn := gtk.NewButtonWithLabel(i18n.T("About Me"))
	aboutBtn.ConnectClicked(w.openAbout)

	sourceHelp := settingsHelp(i18n.T("Browse the app source on GitHub"))
	source := gtk.NewLinkButtonWithLabel(brand.SourceURL, i18n.T("Source Code"))
	source.SetHAlign(gtk.AlignStart)

	ver := gtk.NewLabel("Lamha " + version.String())
	alignStart(ver)
	ver.SetMarginTop(12)
	ver.SetCSSClasses([]string{"dim-label"})

	box.Append(aboutTitle)
	box.Append(aboutHelp)
	box.Append(aboutBtn)
	box.Append(sourceHelp)
	box.Append(source)
	box.Append(ver)
	return box
}

func settingsHeading(text string, top int) *gtk.Label {
	label := gtk.NewLabel(text)
	alignStart(label)
	label.SetCSSClasses([]string{"title-4"})
	if top > 0 {
		label.SetMarginTop(top)
	}
	return label
}

func settingsHelp(text string) *gtk.Label {
	help := gtk.NewLabel(text)
	alignStart(help)
	help.SetWrap(true)
	help.SetCSSClasses([]string{"dim-label"})
	return help
}

func zoomCaption(zoom float64) string {
	return i18n.Tf("Magnifier zoom: %.1f×", prefs.ClampZoom(zoom))
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
