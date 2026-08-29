package ui

import (
	"context"
	"errors"
	"strings"

	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gdkwayland/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

const appID = "io.github.lamha.Lamha"

// PortalParent exports a Wayland handle for desktop-portal dialogs.
func (w *Window) PortalParent(ctx context.Context) (string, func()) {
	return w.exportPortalParent(ctx)
}

func (w *Window) exportPortalParent(ctx context.Context) (string, func()) {
	type result struct {
		parent string
		drop   func()
	}

	done := make(chan result, 1)
	glib.IdleAdd(func() {
		w.dropExportedHandle()
		err := requestWaylandExport(w.window, func(parent, handle string, toplevel *gdkwayland.WaylandToplevel) {
			w.exportedHandle = handle
			w.exportedTop = toplevel
			done <- result{parent: parent, drop: func() {
				glib.IdleAdd(func() { w.dropExportedHandle() })
			}}
		})
		if err != nil {
			done <- result{drop: func() {}}
		}
	})

	select {
	case out := <-done:
		if out.drop == nil {
			return "", func() {}
		}
		return out.parent, out.drop
	case <-ctx.Done():
		return "", func() {}
	}
}

func (w *Window) dropExportedHandle() {
	if w.exportedTop == nil || w.exportedHandle == "" {
		return
	}
	handle := w.exportedHandle
	top := w.exportedTop
	w.exportedHandle = ""
	w.exportedTop = nil
	top.DropExportedHandle(handle)
}

func (w *Window) portalParentIfMapped(ctx context.Context) (string, func()) {
	type mapped struct{ ok bool }
	ready := make(chan mapped, 1)
	glib.IdleAdd(func() {
		ready <- mapped{ok: w.window.Mapped() && w.window.Surface() != nil}
	})
	select {
	case out := <-ready:
		if !out.ok {
			return "", func() {}
		}
	case <-ctx.Done():
		return "", func() {}
	}
	return w.exportPortalParent(ctx)
}

func requestWaylandExport(window *gtk.ApplicationWindow, ready func(parent, handle string, toplevel *gdkwayland.WaylandToplevel)) error {
	surface := window.Surface()
	if surface == nil {
		return errors.New("window is not realized")
	}
	toplevel := waylandToplevel(surface)
	if toplevel == nil {
		return errors.New("window is not a Wayland toplevel")
	}

	toplevel.SetApplicationID(appID)
	if !toplevel.ExportHandle(func(_ *gdkwayland.WaylandToplevel, handle string) {
		parent := handle
		if handle != "" && !strings.HasPrefix(handle, "wayland:") && !strings.HasPrefix(handle, "x11:") {
			parent = "wayland:" + handle
		}
		ready(parent, handle, toplevel)
	}) {
		return errors.New("could not export Wayland window handle")
	}
	return nil
}

func waylandToplevel(surface gdk.Surfacer) *gdkwayland.WaylandToplevel {
	if surface == nil {
		return nil
	}
	if toplevel, ok := surface.(*gdkwayland.WaylandToplevel); ok {
		return toplevel
	}

	object := glib.BaseObject(surface)
	if object == nil {
		return nil
	}
	casted := object.WalkCast(func(obj glib.Objector) bool {
		_, ok := obj.(*gdkwayland.WaylandToplevel)
		return ok
	})
	toplevel, _ := casted.(*gdkwayland.WaylandToplevel)
	return toplevel
}
