// Package portal contains the Linux desktop-portal integrations used by Lamha.
//
// Wayland compositors intentionally do not expose raw screen pixels to regular
// applications. The Screenshot portal is the cross-desktop, consent-based API
// implemented by both GNOME and KDE's portal backends.
package portal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	desktopService                 = "org.freedesktop.portal.Desktop"
	desktopPath    dbus.ObjectPath = "/org/freedesktop/portal/desktop"

	screenshotInterface = "org.freedesktop.portal.Screenshot"
	requestInterface    = "org.freedesktop.portal.Request"
	propertiesInterface = "org.freedesktop.DBus.Properties"
)

// Target describes the content a portal should capture when it supports
// Screenshot interface version 3 or newer.
type Target uint32

const (
	TargetDefault      Target = 0
	TargetScreen       Target = 1
	TargetWindow       Target = 2
	TargetArea         Target = 4
	TargetActiveWindow Target = 8
)

// ScreenshotOptions describes one Screenshot portal request.
type ScreenshotOptions struct {
	// ParentWindow is a portal window identifier. It can be empty when no
	// exported parent handle is available.
	ParentWindow string
	// Interactive asks the portal to present its selection UI.
	Interactive bool
	// Target is used only when the active portal backend advertises it.
	Target Target
}

// ScreenshotResult is returned when the user completes a portal request.
type ScreenshotResult struct {
	URI string
}

// ErrCancelled indicates that the user dismissed the portal's capture UI.
var ErrCancelled = errors.New("screenshot was cancelled")

// ErrDenied is portal response 2: GNOME refuses a silent grab until the app
// has been granted screenshot permission through an interactive request.
var ErrDenied = errors.New("screenshot permission denied")

// Client calls XDG desktop portals on the session bus.
type Client struct {
	connect func() (*dbus.Conn, error)
}

// NewClient creates a client using the user's session bus.
func NewClient() *Client {
	return &Client{connect: dbus.SessionBus}
}

// SilentScreenshot grabs the current view without opening a picker. GNOME
// rejects this until permission has been granted via an interactive request
// parented to a real window.
func (c *Client) SilentScreenshot(ctx context.Context, parentWindow string) (ScreenshotResult, error) {
	return c.Screenshot(ctx, ScreenshotOptions{ParentWindow: parentWindow})
}

// Screenshot asks the desktop portal to take a screenshot. Interactive requests
// can show a picker; silent requests never pass a target for that reason.
func (c *Client) Screenshot(ctx context.Context, input ScreenshotOptions) (ScreenshotResult, error) {
	conn, err := c.connect()
	if err != nil {
		return ScreenshotResult{}, fmt.Errorf("connect to session bus: %w", err)
	}

	options := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(newHandleToken()),
		"modal":        dbus.MakeVariant(input.Interactive),
		"interactive":  dbus.MakeVariant(input.Interactive),
	}
	if input.Interactive && c.targetIsAvailable(ctx, conn, input.Target) {
		options["target"] = dbus.MakeVariant(uint32(input.Target))
	}

	// Subscribe before making the method call. This removes the small race in
	// which a very fast backend could emit Request::Response before a listener
	// had been installed.
	signals := make(chan *dbus.Signal, 8)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)

	match := []dbus.MatchOption{
		dbus.WithMatchInterface(requestInterface),
		dbus.WithMatchMember("Response"),
	}
	if err := conn.AddMatchSignalContext(ctx, match...); err != nil {
		return ScreenshotResult{}, fmt.Errorf("subscribe to portal response: %w", err)
	}
	defer func() { _ = conn.RemoveMatchSignal(match...) }()

	var requestPath dbus.ObjectPath
	call := conn.Object(desktopService, desktopPath).CallWithContext(
		ctx,
		screenshotInterface+".Screenshot",
		0,
		input.ParentWindow,
		options,
	)
	if err := call.Store(&requestPath); err != nil {
		return ScreenshotResult{}, fmt.Errorf("request screenshot from portal: %w", err)
	}

	results, err := waitForResults(ctx, conn, requestPath, signals)
	if err != nil {
		return ScreenshotResult{}, err
	}
	uri, ok := results["uri"].Value().(string)
	if !ok || uri == "" {
		return ScreenshotResult{}, errors.New("portal response did not include a screenshot URI")
	}
	return ScreenshotResult{URI: uri}, nil
}

func (c *Client) targetIsAvailable(ctx context.Context, conn *dbus.Conn, target Target) bool {
	if target == TargetDefault {
		return false
	}

	var value dbus.Variant
	call := conn.Object(desktopService, desktopPath).CallWithContext(
		ctx,
		propertiesInterface+".Get",
		0,
		screenshotInterface,
		"AvailableTargets",
	)
	if err := call.Store(&value); err != nil {
		// AvailableTargets was added in Screenshot portal v3. Older portal
		// implementations still honour interactive selection, so silently use
		// that compatible path.
		return false
	}

	targets, ok := value.Value().(uint32)
	return ok && targets&uint32(target) != 0
}

func waitForResults(ctx context.Context, conn *dbus.Conn, requestPath dbus.ObjectPath, signals <-chan *dbus.Signal) (map[string]dbus.Variant, error) {
	for {
		select {
		case signal, ok := <-signals:
			if !ok {
				return nil, errors.New("session bus connection closed")
			}
			if signal == nil || signal.Path != requestPath {
				continue
			}
			return decodeResults(signal)
		case <-ctx.Done():
			_ = conn.Object(desktopService, requestPath).Call(
				requestInterface+".Close", 0,
			).Err
			return nil, ctx.Err()
		}
	}
}

func decodeResults(signal *dbus.Signal) (map[string]dbus.Variant, error) {
	if signal.Name != requestInterface+".Response" || len(signal.Body) != 2 {
		return nil, errors.New("received malformed portal response")
	}

	response, ok := signal.Body[0].(uint32)
	if !ok {
		return nil, errors.New("received invalid portal response code")
	}
	if response == 1 {
		return nil, ErrCancelled
	}
	if response == 2 {
		return nil, ErrDenied
	}
	if response != 0 {
		return nil, fmt.Errorf("portal failed with response code %d", response)
	}

	results, ok := signal.Body[1].(map[string]dbus.Variant)
	if !ok {
		return nil, errors.New("received invalid portal result data")
	}
	return results, nil
}

func newHandleToken() string {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		// It is only a request correlation token. A predictable fallback does
		// not weaken the portal's permission model.
		return "lamha_request"
	}
	return "lamha_" + hex.EncodeToString(bytes)
}
