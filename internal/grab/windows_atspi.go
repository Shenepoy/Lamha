package grab

import (
	"context"
	"fmt"
	"image"
	"strings"

	"github.com/godbus/dbus/v5"
)

type atspiRef struct {
	Dest string
	Path dbus.ObjectPath
}

func listATSPI(ctx context.Context) ([]WindowFrame, error) {
	session, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}
	var addr string
	if err := session.Object("org.a11y.Bus", "/org/a11y/bus").CallWithContext(ctx, "org.a11y.Bus.GetAddress", 0).Store(&addr); err != nil {
		return nil, fmt.Errorf("AT-SPI bus: %w", err)
	}
	conn, err := dbus.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("AT-SPI dial: %w", err)
	}
	defer conn.Close()
	if err := conn.Auth(nil); err != nil {
		return nil, fmt.Errorf("AT-SPI auth: %w", err)
	}
	if err := conn.Hello(); err != nil {
		return nil, fmt.Errorf("AT-SPI hello: %w", err)
	}

	root := atspiRef{Dest: "org.a11y.atspi.Registry", Path: "/org/a11y/atspi/accessible/root"}
	apps, err := atspiChildren(conn, root)
	if err != nil {
		return nil, err
	}
	var out []WindowFrame
	for _, app := range apps {
		if ctx.Err() != nil {
			break
		}
		kids, err := atspiChildren(conn, app)
		if err != nil {
			continue
		}
		for _, child := range kids {
			role := strings.ToLower(atspiRoleName(conn, child))
			if !atspiWindowRole(role) {
				continue
			}
			bounds, ok := atspiExtents(conn, child)
			if !ok {
				continue
			}
			out = append(out, WindowFrame{
				Title:   atspiName(conn, child),
				Class:   atspiName(conn, app) + " " + role,
				Bounds:  bounds,
				Focused: atspiIsFocused(conn, child),
			})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("AT-SPI found no window frames")
	}
	return out, nil
}

func atspiWindowRole(role string) bool {
	switch role {
	case "frame", "window", "dialog", "alert", "alert dialog", "file chooser", "color chooser":
		return true
	default:
		return false
	}
}

func atspiChildren(conn *dbus.Conn, ref atspiRef) ([]atspiRef, error) {
	obj := conn.Object(ref.Dest, ref.Path)
	var kids []struct {
		Dest string
		Path dbus.ObjectPath
	}
	if err := obj.Call("org.a11y.atspi.Accessible.GetChildren", 0).Store(&kids); err == nil {
		out := make([]atspiRef, 0, len(kids))
		for _, kid := range kids {
			out = append(out, atspiRef{Dest: kid.Dest, Path: kid.Path})
		}
		return out, nil
	}
	var n int32
	if err := obj.Call("org.a11y.atspi.Accessible.GetChildCount", 0).Store(&n); err != nil {
		return nil, err
	}
	out := make([]atspiRef, 0, n)
	for i := int32(0); i < n; i++ {
		var dest string
		var path dbus.ObjectPath
		if err := obj.Call("org.a11y.atspi.Accessible.GetChildAtIndex", 0, i).Store(&dest, &path); err != nil {
			continue
		}
		out = append(out, atspiRef{Dest: dest, Path: path})
	}
	return out, nil
}

func atspiRoleName(conn *dbus.Conn, ref atspiRef) string {
	var role string
	_ = conn.Object(ref.Dest, ref.Path).Call("org.a11y.atspi.Accessible.GetRoleName", 0).Store(&role)
	return role
}

func atspiName(conn *dbus.Conn, ref atspiRef) string {
	var name string
	_ = conn.Object(ref.Dest, ref.Path).Call("org.a11y.atspi.Accessible.GetName", 0).Store(&name)
	return name
}

const (
	atspiStateActive  = 1
	atspiStateFocused = 12
)

func atspiIsFocused(conn *dbus.Conn, ref atspiRef) bool {
	var states []uint32
	if err := conn.Object(ref.Dest, ref.Path).Call("org.a11y.atspi.Accessible.GetState", 0).Store(&states); err != nil {
		return false
	}
	return atspiStateHas(states, atspiStateActive) || atspiStateHas(states, atspiStateFocused)
}

func atspiStateHas(states []uint32, bit uint32) bool {
	idx := bit / 32
	if int(idx) >= len(states) {
		return false
	}
	return states[idx]&(1<<(bit%32)) != 0
}

func atspiExtents(conn *dbus.Conn, ref atspiRef) (image.Rectangle, bool) {
	var x, y, w, h int32
	err := conn.Object(ref.Dest, ref.Path).Call("org.a11y.atspi.Component.GetExtents", 0, int32(0)).Store(&x, &y, &w, &h)
	if err != nil || w < 1 || h < 1 {
		return image.Rectangle{}, false
	}
	return image.Rect(int(x), int(y), int(x+w), int(y+h)), true
}
