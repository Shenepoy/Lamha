package portal

import (
	"context"
	"fmt"
	"log"

	"github.com/godbus/dbus/v5"
)

const (
	shortcutsInterface = "org.freedesktop.portal.GlobalShortcuts"
	ShortcutArea       = "capture-area"
	ShortcutWindow     = "capture-window"
	ShortcutScreen     = "capture-screen"
)

// Shortcut is one global hotkey offered to the desktop portal.
type Shortcut struct {
	ID          string
	Description string
	Trigger     string
}

// DefaultCaptureShortcuts are the system-level screenshot bindings Lamha asks for.
func DefaultCaptureShortcuts() []Shortcut {
	return []Shortcut{
		{ID: ShortcutArea, Description: "Capture area", Trigger: "CTRL+SHIFT+A"},
		{ID: ShortcutWindow, Description: "Capture window", Trigger: "CTRL+SHIFT+W"},
		{ID: ShortcutScreen, Description: "Capture screen", Trigger: "CTRL+SHIFT+S"},
	}
}

// BindGlobalShortcuts asks the portal to register system-wide screenshot keys.
// onActivate is called with the shortcut ID from the GTK-safe caller.
func (c *Client) BindGlobalShortcuts(ctx context.Context, parentWindow string, shortcuts []Shortcut, onActivate func(id string)) error {
	conn, err := c.connect()
	if err != nil {
		return fmt.Errorf("connect to session bus: %w", err)
	}

	signals := make(chan *dbus.Signal, 16)
	conn.Signal(signals)
	if err := conn.AddMatchSignalContext(ctx,
		dbus.WithMatchInterface(requestInterface),
		dbus.WithMatchMember("Response"),
	); err != nil {
		return fmt.Errorf("subscribe to portal response: %w", err)
	}
	if err := conn.AddMatchSignalContext(ctx,
		dbus.WithMatchInterface(shortcutsInterface),
		dbus.WithMatchMember("Activated"),
	); err != nil {
		return fmt.Errorf("subscribe to shortcut activations: %w", err)
	}

	sessionToken := newHandleToken()
	options := map[string]dbus.Variant{
		"handle_token":         dbus.MakeVariant(newHandleToken()),
		"session_handle_token": dbus.MakeVariant(sessionToken),
	}

	var requestPath dbus.ObjectPath
	call := conn.Object(desktopService, desktopPath).CallWithContext(
		ctx,
		shortcutsInterface+".CreateSession",
		0,
		options,
	)
	if err := call.Store(&requestPath); err != nil {
		return fmt.Errorf("create global shortcuts session: %w", err)
	}

	results, err := waitForResults(ctx, conn, requestPath, signals)
	if err != nil {
		return fmt.Errorf("global shortcuts session: %w", err)
	}
	session, err := objectPathOf(results, "session_handle")
	if err != nil {
		return err
	}

	payload := make([]struct {
		ID      string
		Options map[string]dbus.Variant
	}, len(shortcuts))
	for i, shortcut := range shortcuts {
		opts := map[string]dbus.Variant{
			"description": dbus.MakeVariant(shortcut.Description),
		}
		if shortcut.Trigger != "" {
			opts["preferred_trigger"] = dbus.MakeVariant(shortcut.Trigger)
		}
		payload[i] = struct {
			ID      string
			Options map[string]dbus.Variant
		}{ID: shortcut.ID, Options: opts}
	}

	bindOpts := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(newHandleToken()),
	}
	call = conn.Object(desktopService, desktopPath).CallWithContext(
		ctx,
		shortcutsInterface+".BindShortcuts",
		0,
		session,
		payload,
		parentWindow,
		bindOpts,
	)
	if err := call.Store(&requestPath); err != nil {
		return fmt.Errorf("bind global shortcuts: %w", err)
	}
	if _, err := waitForResults(ctx, conn, requestPath, signals); err != nil {
		return fmt.Errorf("bind global shortcuts: %w", err)
	}

	go func() {
		for signal := range signals {
			if signal == nil || signal.Name != shortcutsInterface+".Activated" || len(signal.Body) < 2 {
				continue
			}
			id, ok := signal.Body[1].(string)
			if !ok || id == "" {
				continue
			}
			log.Printf("global shortcut %s", id)
			onActivate(id)
		}
	}()

	log.Printf("global shortcuts bound on session %s", session)
	return nil
}

func objectPathOf(results map[string]dbus.Variant, key string) (dbus.ObjectPath, error) {
	value, ok := results[key]
	if !ok {
		return "", fmt.Errorf("portal response missing %s", key)
	}
	switch typed := value.Value().(type) {
	case dbus.ObjectPath:
		return typed, nil
	case string:
		if typed == "" {
			return "", fmt.Errorf("portal returned an empty %s", key)
		}
		return dbus.ObjectPath(typed), nil
	default:
		return "", fmt.Errorf("portal returned unexpected %s type %T", key, typed)
	}
}
