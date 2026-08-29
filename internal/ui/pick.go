package ui

import (
	"context"
	"errors"
	"image"
	"log"
	"os"
	"time"

	"github.com/diamondburned/gotk4/pkg/core/glib"

	"github.com/lamha-app/lamha/internal/grab"
	"github.com/lamha-app/lamha/internal/i18n"
	"github.com/lamha-app/lamha/internal/portal"
)

type pickerFrame struct {
	source grab.WindowFrame
	bounds image.Rectangle
}

// StartWindowPick freezes the desktop, then lets the user click a window without activating it.
func (w *Window) StartWindowPick() {
}

func (w *Window) startWindowCapture(ctx context.Context, delay time.Duration, mon image.Rectangle) {
	w.hideForCapture()

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

	frames, listErr := grab.ListWindows(ctx)
	if listErr != nil {
		log.Printf("window list: %v", listErr)
	} else {
		log.Printf("window list: %d frames", len(frames))
	}

	staging, err := w.grabSilent(ctx)
	if err != nil {
		w.captureFailed(err)
		return
	}
	if size, sizeErr := grab.PNGSize(staging); sizeErr == nil {
		log.Printf("window grab size=%dx%d", size.X, size.Y)
	}

	glib.IdleAdd(func() {
		w.setBusy(false, "")
		log.Printf("opening capture overlay")
		w.openCaptureOverlayOn(CaptureWindow, staging, frames, mon)
	})
}

func (w *Window) hideForCapture() {
	done := make(chan struct{})
	glib.IdleAdd(func() {
		if w.overlay != nil {
			w.overlay.close(false)
		}
		w.hidePortalHostLocked()
		w.window.SetVisible(false)
		close(done)
	})
	<-done
	time.Sleep(200 * time.Millisecond)
}

func (w *Window) grabSilent(ctx context.Context) (string, error) {
	log.Printf("trying silent compositor grab")
	staging, err := grab.Fast(ctx)
	if err == nil && !grab.TooSmall(staging) {
		return staging, nil
	}
	if err != nil {
		log.Printf("compositor grab unavailable, using desktop portal: %v", err)
	} else {
		log.Printf("compositor grab was too small; using desktop portal")
		os.Remove(staging)
	}
	glib.IdleAdd(func() {
		w.status.SetText(i18n.T("Requesting screenshot access…"))
	})

	parent := ""
	if w.windowMapped() {
		var drop func()
		parent, drop = w.exportPortalParent(ctx)
		defer drop()
	}
	log.Printf("portal parent=%q", parent)

	staging, err = grab.ViaPortal(ctx, portal.ScreenshotOptions{ParentWindow: parent})
	if err == nil && !grab.TooSmall(staging) {
		return staging, nil
	}
	if err != nil && errors.Is(err, portal.ErrCancelled) {
		return "", err
	}
	if err == nil {
		log.Printf("silent portal grab was too small")
		os.Remove(staging)
	}

	log.Printf("silent portal failed; asking GNOME for permission: %v", err)
	glib.IdleAdd(func() {
		w.status.SetText(i18n.T("GNOME needs one-time permission. Allow the system screenshot dialog."))
	})
	w.presentForPortal()
	parent, drop := w.exportPortalParent(ctx)
	defer drop()
	log.Printf("portal parent=%q", parent)
	staging, err = grab.ViaPortal(ctx, portal.ScreenshotOptions{
		ParentWindow: parent,
		Interactive:  true,
	})
	if !w.restoreAfterCapture {
		glib.IdleAdd(func() { w.window.SetVisible(false) })
	}
	if err != nil {
		log.Printf("portal grab failed: %v", err)
		return "", err
	}
	if grab.TooSmall(staging) {
		os.Remove(staging)
		return "", errors.New("screenshot was empty")
	}
	return staging, nil
}

func (w *Window) hidePortalHost() {
	glib.IdleAdd(func() { w.hidePortalHostLocked() })
}

func (w *Window) hidePortalHostLocked() {
	if w.portalHost != nil {
		w.portalHost.SetVisible(false)
	}
}

func (w *Window) windowMapped() bool {
	ready := make(chan bool, 1)
	glib.IdleAdd(func() {
		ready <- w.window.Mapped() && w.window.Surface() != nil
	})
	return <-ready
}

func mapPickerFrames(frames []grab.WindowFrame, mon image.Rectangle) []pickerFrame {
	out := make([]pickerFrame, 0, len(frames))
	for _, frame := range frames {
		r := frame.Bounds
		if !mon.Empty() {
			r = r.Intersect(mon).Sub(mon.Min)
		}
		if r.Dx() >= 32 && r.Dy() >= 32 {
			out = append(out, pickerFrame{source: frame, bounds: r})
		}
	}
	return out
}

func pickerToScreen(r image.Rectangle, mon image.Rectangle) image.Rectangle {
	if mon.Empty() {
		return r
	}
	return r.Add(mon.Min)
}
