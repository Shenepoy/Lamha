package ui

import (
	"bytes"
	"fmt"
	"log"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

// copyImageFile is used by interactive UI actions. GTK associates this
// clipboard request with the current input event, which is the most reliable
// path for an explicit Copy button or menu action.
func copyImageFile(path string) error {
	texture, err := gdk.NewTextureFromFilename(path)
	if err != nil {
		return fmt.Errorf("load image for clipboard: %w", err)
	}
	display := gdk.DisplayGetDefault()
	if display == nil {
		return fmt.Errorf("no display is available for the clipboard")
	}
	display.Clipboard().SetTexture(texture)
	return nil
}

// copyImageFileForCapture is used after an asynchronous capture. It cannot
// rely on a GTK input serial because the capture may have started from a
// global shortcut while Lamha was hidden, so prefer wl-copy's data-control
// protocol there.
func copyImageFileForCapture(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read image for clipboard: %w", err)
	}
	if len(data) == 0 {
		return fmt.Errorf("image for clipboard is empty")
	}
	return copyClipboardBytes(data, imageClipboardMIME(path))
}

func copyText(text string) error {
	display := gdk.DisplayGetDefault()
	if display == nil {
		return fmt.Errorf("no display is available for the clipboard")
	}
	display.Clipboard().SetText(text)
	return nil
}

func imageClipboardMIME(path string) string {
	if mimeType := mime.TypeByExtension(filepath.Ext(path)); strings.HasPrefix(mimeType, "image/") {
		return mimeType
	}
	return "image/png"
}

func copyClipboardBytes(data []byte, mimeType string) error {
	if isWaylandClipboard() {
		if err := copyWithWaylandCLI(data, mimeType); err == nil {
			return nil
		} else {
			log.Printf("clipboard: wl-copy unavailable or failed: %v", err)
		}
	}

	if err := copyWithGDK(data, mimeType); err != nil {
		return err
	}
	return nil
}

// isWaylandClipboard reports whether the current desktop exposes a Wayland
// clipboard. Checking both the live GDK display and the environment keeps the
// helper useful before a window has been mapped.
func isWaylandClipboard() bool {
	if display := gdk.DisplayGetDefault(); display != nil {
		if strings.HasPrefix(strings.ToLower(display.Name()), "wayland") {
			return true
		}
	}
	return os.Getenv("WAYLAND_DISPLAY") != ""
}

// copyWithWaylandCLI uses the Wayland data-control protocol through wl-copy.
// Unlike wl_data_device.set_selection, data-control does not require a recent
// pointer/keyboard serial, so captures started from a global shortcut can be
// copied reliably while Lamha is hidden.
func copyWithWaylandCLI(data []byte, mimeType string) error {
	path, err := exec.LookPath("wl-copy")
	if err != nil {
		return fmt.Errorf("find wl-copy: %w", err)
	}

	command := exec.Command(path, "--type", mimeType)
	command.Stdin = bytes.NewReader(data)
	if output, err := command.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(output))
		if message != "" {
			return fmt.Errorf("run wl-copy: %w (%s)", err, message)
		}
		return fmt.Errorf("run wl-copy: %w", err)
	}
	return nil
}

func copyWithGDK(data []byte, mimeType string) error {
	display := gdk.DisplayGetDefault()
	if display == nil {
		return fmt.Errorf("no display is available for the clipboard")
	}

	clipboard := display.Clipboard()
	providers := []*gdk.ContentProvider{
		gdk.NewContentProviderForBytes(mimeType, glib.NewBytes(data)),
	}
	if mimeType == "text/plain;charset=utf-8" {
		providers = append(providers, gdk.NewContentProviderForBytes("text/plain", glib.NewBytes(data)))
	}
	provider := providers[0]
	if len(providers) > 1 {
		provider = gdk.NewContentProviderUnion(providers)
	}
	if !clipboard.SetContent(provider) {
		return fmt.Errorf("GDK rejected %s clipboard content", mimeType)
	}
	return nil
}
