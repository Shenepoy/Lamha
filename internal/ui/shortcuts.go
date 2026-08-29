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
	win.SetTransientFor(&w.window.Window)
	win.SetModal(true)
	win.SetDefaultSize(520, 640)
	win.SetHideOnClose(true)
	win.ConnectCloseRequest(func() bool {
		w.shortcuts = nil
		win.Destroy()
		return true
	})

	header := gtk.NewHeaderBar()
	resetAll := gtk.NewButtonWithLabel(i18n.T("Reset all"))
	header.PackStart(resetAll)
	win.SetTitlebar(header)

	status := gtk.NewLabel(i18n.T("Click a shortcut, then press the new keys. Backspace clears it."))
	status.SetWrap(true)
	alignStart(status)
	status.SetCSSClasses([]string{"dim-label"})
	status.SetMarginStart(18)
	status.SetMarginEnd(18)
	status.SetMarginTop(12)

	list := gtk.NewBox(gtk.OrientationVertical, 4)
	list.SetMarginStart(18)
	list.SetMarginEnd(18)
	list.SetMarginBottom(18)

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

	for _, group := range keys.Groups() {
		title := gtk.NewLabel(i18n.T(group.Title))
		alignStart(title)
		title.SetCSSClasses([]string{"title-4"})
		title.SetMarginTop(14)
		title.SetMarginBottom(4)
		list.Append(title)

		for _, binding := range group.Bindings {
			binding := binding
			line := gtk.NewBox(gtk.OrientationHorizontal, 12)
			name := gtk.NewLabel(i18n.T(binding.Label))
			alignStart(name)
			name.SetHExpand(true)
			line.Append(name)

			button := gtk.NewButton()
			button.SetSizeRequest(168, -1)
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
			line.Append(button)

			reset := gtk.NewButtonWithLabel(i18n.T("Reset"))
			reset.ConnectClicked(func() {
				recording = ""
				item, _ := keys.Lookup(binding.ID)
				setAccel(binding.ID, item.Default)
			})
			line.Append(reset)

			list.Append(line)
			rows = append(rows, row{binding: binding, button: button})
		}
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
	scroll.SetChild(list)
	scroll.SetVExpand(true)

	root := gtk.NewBox(gtk.OrientationVertical, 0)
	root.Append(status)
	root.Append(scroll)
	win.SetChild(root)

	w.shortcuts = win
	win.Present()
}
