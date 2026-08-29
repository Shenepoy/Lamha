package ui

import (
	"context"
	"errors"
	"strings"
	"time"

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
	return w.portalParent(ctx, false)
}

func (w *Window) portalParent(ctx context.Context, forceHost bool) (string, func()) {
	type mapped struct {
		useMain bool
		ok      bool
	}
	ready := make(chan mapped, 1)
	glib.IdleAdd(func() {
		mainOK := w.window.Mapped() && w.window.Surface() != nil
		ready <- mapped{useMain: mainOK && !forceHost, ok: mainOK}
	})
	select {
	case out := <-ready:
		if out.useMain {
			return w.exportPortalParent(ctx)
		}
	case <-ctx.Done():
		return "", func() {}
	}
	return w.exportPortalHost(ctx)
}

func (w *Window) exportPortalHost(ctx context.Context) (string, func()) {
	host, err := w.waitPortalHost(ctx)
	if err != nil || host == nil {
		return "", func() {}
	}

	type result struct {
		parent string
		drop   func()
	}
	done := make(chan result, 1)
	glib.IdleAdd(func() {
		if host.Surface() == nil {
			done <- result{drop: func() {}}
			return
		}
		w.dropExportedHandle()
		err := requestWaylandExportSurface(host.Surface(), func(parent, handle string, toplevel *gdkwayland.WaylandToplevel) {
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

func (w *Window) waitPortalHost(ctx context.Context) (*gtk.Window, error) {
	ready := make(chan *gtk.Window, 1)
	glib.IdleAdd(func() {
		if w.portalHost != nil && w.portalHost.Mapped() && w.portalHost.Surface() != nil {
			ready <- w.portalHost
			return
		}
		if w.portalHost == nil {
			win := gtk.NewWindow()
			if app := w.window.Application(); app != nil {
				win.SetApplication(app)
			}
			win.SetTitle("Lamha")
			win.SetDecorated(false)
			win.SetResizable(false)
			win.SetDeletable(false)
			win.SetDefaultSize(1, 1)
			w.portalHost = win
		}
		host := w.portalHost
		host.ConnectMap(func() {
			select {
			case ready <- host:
			default:
			}
		})
		host.Present()
	})
	select {
	case host := <-ready:
		return host, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(2 * time.Second):
		return nil, errors.New("portal host did not map")
	}
}

func (w *Window) presentForPortal() {
	ready := make(chan struct{}, 1)
	glib.IdleAdd(func() {
		if w.window.Mapped() && w.window.Surface() != nil {
			ready <- struct{}{}
		}
		w.window.ConnectMap(func() {
			select {
			case ready <- struct{}{}:
			default:
			}
		})
		w.window.Present()
	})
	select {
	case <-ready:
	case <-time.After(2 * time.Second):
	}
}

func requestWaylandExport(window *gtk.ApplicationWindow, ready func(parent, handle string, toplevel *gdkwayland.WaylandToplevel)) error {
	if window == nil {
		return errors.New("window is not realized")
	}
	return requestWaylandExportSurface(window.Surface(), ready)
}

func requestWaylandExportSurface(surface gdk.Surfacer, ready func(parent, handle string, toplevel *gdkwayland.WaylandToplevel)) error {
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
