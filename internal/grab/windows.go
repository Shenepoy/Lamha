package grab

import (
	"context"
	"errors"
	"image"
	"log"
	"os"
	"strings"
	"time"
)

// WindowFrame is a top-level window in screen pixels.
type WindowFrame struct {
	Title   string
	Class   string
	PID     int
	Bounds  image.Rectangle
	Focused bool
}

var errNoFocusedWindow = errors.New("no focused window")

// FocusedWindow returns the top-level window that currently has input focus.
// Lamha's own windows are ignored.
func FocusedWindow(ctx context.Context) (WindowFrame, error) {
	frames, err := ListWindows(ctx)
	if err != nil {
		return WindowFrame{}, err
	}
	for _, frame := range frames {
		if frame.Focused {
			return frame, nil
		}
	}
	return WindowFrame{}, errNoFocusedWindow
}

const windowListTimeout = 1500 * time.Millisecond

// ListWindows returns visible top-level windows. Call this before the overlay
// covers the desktop. GNOME Introspect is often denied; AT-SPI and optional
// shell extensions are tried next.
func ListWindows(ctx context.Context) ([]WindowFrame, error) {
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > windowListTimeout {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, windowListTimeout)
		defer cancel()
	}

	var errs []error
	for _, list := range []func(context.Context) ([]WindowFrame, error){
		listGnomeIntrospect,
		listGnomeWindowCalls,
		listATSPI,
	} {
		frames, err := list(ctx)
		if err != nil {
			log.Printf("lamha windows: %v", err)
			errs = append(errs, err)
			continue
		}
		if cleaned := filterWindows(frames); len(cleaned) > 0 {
			return cleaned, nil
		}
	}
	if len(errs) == 0 {
		return nil, errors.New("no window list backend returned frames")
	}
	return nil, errors.Join(errs...)
}

// MapToImage converts a screen-space window rect onto a screenshot.
// If the grab is a single monitor, screen origin is subtracted.
func MapToImage(screen image.Rectangle, img image.Point, monitor image.Rectangle) image.Rectangle {
	r := screen
	if !monitor.Empty() && img.X == monitor.Dx() && img.Y == monitor.Dy() {
		r = r.Sub(monitor.Min)
	}
	return r.Intersect(image.Rect(0, 0, img.X, img.Y))
}

// FrameAt returns the smallest window containing p.
func FrameAt(frames []image.Rectangle, p image.Point) image.Rectangle {
	best := image.Rectangle{}
	bestArea := 0
	for _, frame := range frames {
		if !p.In(frame) {
			continue
		}
		area := frame.Dx() * frame.Dy()
		if best.Empty() || area < bestArea {
			best = frame
			bestArea = area
		}
	}
	return best
}

func filterWindows(frames []WindowFrame) []WindowFrame {
	self := os.Getpid()
	out := frames[:0]
	for _, frame := range frames {
		if frame.Bounds.Dx() < 32 || frame.Bounds.Dy() < 32 {
			continue
		}
		if frame.PID == self && self > 0 {
			continue
		}
		class := strings.ToLower(frame.Class + " " + frame.Title)
		if strings.Contains(class, "lamha") {
			continue
		}
		out = append(out, frame)
	}
	return out
}
