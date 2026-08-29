package ui

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/i18n"
)

func applyAppDirection() {
	if i18n.RTL() {
		gtk.WidgetSetDefaultDirection(gtk.TextDirRTL)
	}
}

func applyDirection(widget *gtk.Widget) {
	if i18n.RTL() {
		widget.SetDirection(gtk.TextDirRTL)
	}
}

func alignStart(label *gtk.Label) {
	label.SetHAlign(gtk.AlignStart)
	if i18n.RTL() {
		label.SetXAlign(1)
		return
	}
	label.SetXAlign(0)
}
