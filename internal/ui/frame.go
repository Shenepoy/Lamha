package ui

import "github.com/diamondburned/gotk4/pkg/core/glib"

// redrawPump coalesces many pointer events into one GTK frame.
type redrawPump struct {
	pending bool
}

func (p *redrawPump) request(draw func()) {
	if p == nil || draw == nil || p.pending {
		return
	}
	p.pending = true
	glib.IdleAdd(func() {
		p.pending = false
		draw()
	})
}
