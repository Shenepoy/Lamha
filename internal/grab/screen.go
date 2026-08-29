// Package grab takes a fullscreen snapshot so Lamha can show its own overlay.
// Compositor pickers are never used. The XDG portal is only a silent fallback
// and must be called while Lamha still has a visible window.
package grab

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/lamha-app/lamha/internal/capture"
	"github.com/lamha-app/lamha/internal/portal"
)

const fastTimeout = 2 * time.Second

// Fast tries compositor screenshot services that do not show a picker.
func Fast(ctx context.Context) (string, error) {
	return runBackends(ctx, fastTimeout, fastBackends())
}

// ViaPortal asks the screenshot portal. Silent requests do not show a picker.
// Interactive is GNOME's one-time permission path and may show a system dialog.
func ViaPortal(ctx context.Context, opts portal.ScreenshotOptions) (string, error) {
	timeout := 20 * time.Second
	name := "silent portal"
	if opts.Interactive {
		timeout = 2 * time.Minute
		name = "permission portal"
	}
	return runBackends(ctx, timeout, []backend{{
		name: name,
		grab: func(ctx context.Context, dest string) error {
			return portalToFile(ctx, dest, opts)
		},
	}})
}

func hardTimeout(limit time.Duration, fn func() error) error {
	if limit <= 0 {
		return fn()
	}
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		return err
	case <-time.After(limit):
		return fmt.Errorf("timed out after %s: %w", limit, context.DeadlineExceeded)
	}
}

func runBackends(ctx context.Context, perTry time.Duration, list []backend) (string, error) {
	dest, err := os.CreateTemp("", "lamha-grab-*.png")
	if err != nil {
		return "", fmt.Errorf("create staging capture: %w", err)
	}
	destPath := dest.Name()
	if err := dest.Close(); err != nil {
		os.Remove(destPath)
		return "", err
	}

	var errs []error
	for _, backend := range list {
		tryCtx := ctx
		cancel := func() {}
		if perTry > 0 {
			tryCtx, cancel = context.WithTimeout(ctx, perTry)
		}
		err := hardTimeout(perTry, func() error {
			return backend.grab(tryCtx, destPath)
		})
		cancel()
		if err != nil {
			log.Printf("lamha grab %s: %v", backend.name, err)
			errs = append(errs, fmt.Errorf("%s: %w", backend.name, err))
			continue
		}
		if fileNonEmpty(destPath) {
			return destPath, nil
		}
		log.Printf("lamha grab %s: empty file", backend.name)
		errs = append(errs, fmt.Errorf("%s: wrote an empty file", backend.name))
	}

	os.Remove(destPath)
	if len(errs) == 0 {
		return "", errors.New("no screenshot backend is available")
	}
	return "", fmt.Errorf("could not grab the screen: %w", errors.Join(errs...))
}

type backend struct {
	name string
	grab func(context.Context, string) error
}

func fastBackends() []backend {
	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP") + ":" + os.Getenv("DESKTOP_SESSION"))
	gnome := []backend{
		{name: "GNOME Shell", grab: gnomeShellScreenshot},
		{name: "gnome-screenshot", grab: gnomeScreenshotCLI},
		{name: "KWin", grab: kwinScreenshot},
		{name: "Spectacle", grab: spectacleScreenshot},
	}
	plasma := []backend{
		{name: "KWin", grab: kwinScreenshot},
		{name: "Spectacle", grab: spectacleScreenshot},
		{name: "GNOME Shell", grab: gnomeShellScreenshot},
	}
	if strings.Contains(desktop, "kde") || strings.Contains(desktop, "plasma") {
		return plasma
	}
	return gnome
}

func gnomeShellScreenshot(ctx context.Context, dest string) error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return err
	}
	var ok bool
	var used string
	call := conn.Object("org.gnome.Shell.Screenshot", "/org/gnome/Shell/Screenshot").CallWithContext(
		ctx,
		"org.gnome.Shell.Screenshot.Screenshot",
		0,
		false,
		false,
		dest,
	)
	if err := call.Store(&ok, &used); err != nil {
		return err
	}
	if !ok {
		return errors.New("GNOME Shell refused the screenshot")
	}
	if used != "" && used != dest {
		return copyFile(used, dest)
	}
	return nil
}

func kwinScreenshot(ctx context.Context, dest string) error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return err
	}
	obj := conn.Object("org.kde.KWin", "/Screenshot")
	var filename string
	call := obj.CallWithContext(ctx, "org.kde.kwin.Screenshot.screenshotFullscreen", 0, false)
	if err := call.Store(&filename); err != nil {
		call = obj.CallWithContext(ctx, "org.kde.kwin.Screenshot.screenshotFullscreen", 0)
		if err := call.Store(&filename); err != nil {
			return err
		}
	}
	if filename == "" {
		return errors.New("KWin returned an empty screenshot path")
	}
	if filename == dest {
		return nil
	}
	return copyFile(filename, dest)
}

func gnomeScreenshotCLI(ctx context.Context, dest string) error {
	bin, err := exec.LookPath("gnome-screenshot")
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, bin, "-f", dest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w (%s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func spectacleScreenshot(ctx context.Context, dest string) error {
	spectacle, err := exec.LookPath("spectacle")
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, spectacle, "-b", "-n", "-f", "-o", dest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w (%s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func portalToFile(ctx context.Context, dest string, opts portal.ScreenshotOptions) error {
	result, err := portal.NewClient().Screenshot(ctx, opts)
	if err != nil {
		return err
	}
	staged, err := capture.CopyURIToTemp(result.URI)
	if err != nil {
		return err
	}
	defer os.Remove(staged)
	return copyFile(staged, dest)
}

func copyFile(source, dest string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func fileNonEmpty(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Size() > 0 && !info.IsDir()
}
