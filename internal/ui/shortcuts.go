package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/brand"
	"github.com/lamha-app/lamha/internal/hotkeys"
	"github.com/lamha-app/lamha/internal/i18n"
	"github.com/lamha-app/lamha/internal/keys"
)

func (w *Window) openShortcutSettings() {
	if w.shortcuts != nil {
		w.shortcuts.Present()
		return
	}

	win := gtk.NewWindow()
	if app := w.window.Application(); app != nil {
		win.SetApplication(app)
	}
	win.SetTitle(i18n.T("Keyboard shortcuts"))
	win.SetIconName(brand.Name)
	applyDirection(&win.Widget)
	if w.settingsWin != nil {
		win.SetTransientFor(w.settingsWin)
	} else {
		win.SetTransientFor(&w.window.Window)
	}
	win.SetModal(true)
	win.SetDefaultSize(540, 700)
	win.SetHideOnClose(true)
	win.ConnectCloseRequest(func() bool {
		w.shortcuts = nil
		win.Destroy()
		return true
	})

	header := gtk.NewHeaderBar()
	resetAll := gtk.NewButtonWithLabel(i18n.T("Reset all"))
	resetAll.AddCSSClass("flat")
	header.PackStart(resetAll)
	win.SetTitlebar(header)

	ensurePrefsCSS()
	status := prefsBanner(i18n.T("Click a shortcut, then press the new keys. Backspace clears it."))

	type row struct {
		binding keys.Binding
		button  *gtk.Button
	}
	var rows []row
	var recording keys.ID

	refresh := func() {
		for _, item := range rows {
			label := accelLabel(keys.Current().Accel(item.binding.ID))
			if label == "" {
				label = i18n.T("Disabled")
			}
			if recording == item.binding.ID {
				label = i18n.T("Press a key…")
				item.button.AddCSSClass("suggested-action")
			} else {
				item.button.RemoveCSSClass("suggested-action")
			}
			item.button.SetLabel(label)
		}
	}

	applySystem := func() {
		if err := hotkeys.InstallGNOME(keys.Current().GNOMEAccels()); err != nil {
			status.SetText(err.Error())
		}
	}

	setAccel := func(id keys.ID, accel string) {
		if err := keys.Current().Set(id, accel); err != nil {
			status.SetText(err.Error())
			return
		}
		item, _ := keys.Lookup(id)
		if accel == "" {
			status.SetText(i18n.Tf("%s is disabled.", i18n.T(item.Label)))
		} else {
			status.SetText(i18n.Tf("%s is now %s.", i18n.T(item.Label), accelLabel(accel)))
		}
		if item.System {
			applySystem()
		}
		refresh()
	}

	page := prefsPage()
	page.Append(status)

	for _, group := range keys.Groups() {
		wrap, card := prefsGroup(i18n.T(group.Title), "")
		visible := 0
		for _, binding := range group.Bindings {
			if binding.ID == keys.CaptureWindow {
				continue
			}
			binding := binding
			button := gtk.NewButton()
			button.AddCSSClass("lamha-key")
			button.AddCSSClass("monospace")
			button.SetVAlign(gtk.AlignCenter)
			button.SetHAlign(gtk.AlignEnd)
			button.ConnectClicked(func() {
				if recording == binding.ID {
					recording = ""
					status.SetText(i18n.T("Click a shortcut, then press the new keys. Backspace clears it."))
					refresh()
					return
				}
				recording = binding.ID
				status.SetText(i18n.Tf("Press a shortcut for %s. Escape cancels.", i18n.T(binding.Label)))
				refresh()
			})

			reset := gtk.NewButtonWithLabel(i18n.T("Reset"))
			reset.AddCSSClass("flat")
			reset.SetVAlign(gtk.AlignCenter)
			reset.ConnectClicked(func() {
				recording = ""
				item, _ := keys.Lookup(binding.ID)
				setAccel(binding.ID, item.Default)
			})

			actions := gtk.NewBox(gtk.OrientationHorizontal, 6)
			actions.SetVAlign(gtk.AlignCenter)
			actions.Append(button)
			actions.Append(reset)

			card.Append(prefsRow(i18n.T(binding.Label), "", actions))
			rows = append(rows, row{binding: binding, button: button})
			visible++
		}
		if visible == 0 {
			continue
		}
		page.Append(wrap)
	}

	resetAll.ConnectClicked(func() {
		recording = ""
		if err := keys.Current().ResetAll(); err != nil {
			status.SetText(err.Error())
			return
		}
		applySystem()
		status.SetText(i18n.T("All shortcuts restored to defaults."))
		refresh()
	})

	keysCtl := gtk.NewEventControllerKey()
	keysCtl.SetPropagationPhase(gtk.PhaseCapture)
	keysCtl.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		if recording == "" {
			return false
		}
		if keyval == gdk.KEY_Escape {
			recording = ""
			status.SetText(i18n.T("Click a shortcut, then press the new keys. Backspace clears it."))
			refresh()
			return true
		}
		if keyval == gdk.KEY_BackSpace || keyval == gdk.KEY_Delete {
			id := recording
			recording = ""
			setAccel(id, "")
			return true
		}
		accel, ok := accelFromEvent(keyval, state)
		if !ok {
			return true
		}
		id := recording
		recording = ""
		setAccel(id, accel)
		return true
	})
	win.AddController(keysCtl)

	refresh()

	scroll := gtk.NewScrolledWindow()
	scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scroll.SetChild(page)
	scroll.SetVExpand(true)
	win.SetChild(scroll)

	w.shortcuts = win
	win.Present()
}
