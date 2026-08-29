package portal

import (
	"context"
	"fmt"

	"github.com/godbus/dbus/v5"
)

const backgroundInterface = "org.freedesktop.portal.Background"

// RequestBackground asks the desktop to let Lamha keep running with no window.
func (c *Client) RequestBackground(ctx context.Context, parentWindow string) error {
	conn, err := c.connect()
	if err != nil {
		return fmt.Errorf("connect to session bus: %w", err)
	}

	signals := make(chan *dbus.Signal, 8)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)

	match := []dbus.MatchOption{
		dbus.WithMatchInterface(requestInterface),
		dbus.WithMatchMember("Response"),
	}
	if err := conn.AddMatchSignalContext(ctx, match...); err != nil {
		return fmt.Errorf("subscribe to portal response: %w", err)
	}
	defer func() { _ = conn.RemoveMatchSignal(match...) }()

	options := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(newHandleToken()),
		"reason":       dbus.MakeVariant("Stay ready to take screenshots from the tray and system shortcuts."),
		"autostart":    dbus.MakeVariant(false),
	}

	var requestPath dbus.ObjectPath
	call := conn.Object(desktopService, desktopPath).CallWithContext(
		ctx,
		backgroundInterface+".RequestBackground",
		0,
		parentWindow,
		options,
	)
	if err := call.Store(&requestPath); err != nil {
		return fmt.Errorf("request background stay: %w", err)
	}
	if _, err := waitForResults(ctx, conn, requestPath, signals); err != nil {
		return fmt.Errorf("request background stay: %w", err)
	}
	return nil
}
