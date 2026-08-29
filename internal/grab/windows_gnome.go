package grab

import (
	"context"
	"encoding/json"
	"fmt"
	"image"

	"github.com/godbus/dbus/v5"
)

func listGnomeIntrospect(ctx context.Context) ([]WindowFrame, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}
	var raw map[uint64]map[string]dbus.Variant
	call := conn.Object("org.gnome.Shell.Introspect", "/org/gnome/Shell/Introspect").CallWithContext(
		ctx,
		"org.gnome.Shell.Introspect.GetWindows",
		0,
	)
	if err := call.Store(&raw); err != nil {
		call = conn.Object("org.gnome.Shell", "/org/gnome/Shell/Introspect").CallWithContext(
			ctx,
			"org.gnome.Shell.Introspect.GetWindows",
			0,
		)
		if err := call.Store(&raw); err != nil {
			return nil, fmt.Errorf("GNOME Introspect: %w", err)
		}
	}
	out := make([]WindowFrame, 0, len(raw))
	for _, props := range raw {
		if variantBool(props, "hidden") {
			continue
		}
		if inWS, ok := props["in-current-workspace"]; ok {
			if v, ok := inWS.Value().(bool); ok && !v {
				continue
			}
		}
		w, h := variantInt(props, "width"), variantInt(props, "height")
		x, y := variantInt(props, "x"), variantInt(props, "y")
		if w < 1 || h < 1 {
			continue
		}
		out = append(out, WindowFrame{
			Title:   variantString(props, "title"),
			Class:   variantString(props, "wm-class"),
			PID:     variantInt(props, "pid"),
			Bounds:  image.Rect(x, y, x+w, y+h),
			Focused: variantBool(props, "focus") || variantBool(props, "is-focused") || variantBool(props, "has-focus"),
		})
	}
	return out, nil
}

func listGnomeWindowCalls(ctx context.Context) ([]WindowFrame, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}
	var payload string
	call := conn.Object("org.gnome.Shell", "/org/gnome/Shell/Extensions/Windows").CallWithContext(
		ctx,
		"org.gnome.Shell.Extensions.Windows.List",
		0,
	)
	if err := call.Store(&payload); err != nil {
		return nil, fmt.Errorf("window-calls: %w", err)
	}
	var rows []struct {
		Title               string `json:"title"`
		WMClass             string `json:"wm_class"`
		PID                 int    `json:"pid"`
		X                   int    `json:"x"`
		Y                   int    `json:"y"`
		Width               int    `json:"width"`
		Height              int    `json:"height"`
		InCurrentWorkspace *bool `json:"in_current_workspace"`
		WindowType         int   `json:"window_type"`
		Focus              bool  `json:"focus"`
		HasFocus           *bool `json:"has_focus"`
	}
	if err := json.Unmarshal([]byte(payload), &rows); err != nil {
		return nil, fmt.Errorf("window-calls json: %w", err)
	}
	out := make([]WindowFrame, 0, len(rows))
	for _, row := range rows {
		if row.InCurrentWorkspace != nil && !*row.InCurrentWorkspace {
			continue
		}
		if row.WindowType != 0 {
			continue
		}
		if row.Width < 1 || row.Height < 1 {
			continue
		}
		out = append(out, WindowFrame{
			Title:   row.Title,
			Class:   row.WMClass,
			PID:     row.PID,
			Bounds:  image.Rect(row.X, row.Y, row.X+row.Width, row.Y+row.Height),
			Focused: row.Focus || (row.HasFocus != nil && *row.HasFocus),
		})
	}
	return out, nil
}

func variantString(props map[string]dbus.Variant, key string) string {
	v, ok := props[key]
	if !ok {
		return ""
	}
	s, _ := v.Value().(string)
	return s
}

func variantBool(props map[string]dbus.Variant, key string) bool {
	v, ok := props[key]
	if !ok {
		return false
	}
	b, _ := v.Value().(bool)
	return b
}

func variantInt(props map[string]dbus.Variant, key string) int {
	v, ok := props[key]
	if !ok {
		return 0
	}
	switch n := v.Value().(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case uint:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	default:
		return 0
	}
}
