package indicator

import (
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/lamha-app/lamha/internal/brand"
	"github.com/lamha-app/lamha/internal/i18n"
)

// How long to wait after a tray-menu click so GNOME/KDE can unmap the popup
// and paint a clean frame before the screenshot.
const menuDismissWait = 350 * time.Millisecond

const (
	menuCaptureArea   int32 = 1
	menuCaptureWindow int32 = 2
	menuCaptureScreen int32 = 3
	menuOpen          int32 = 5
	menuQuit          int32 = 6
)

type menuNode struct {
	ID       int32
	Props    map[string]dbus.Variant
	Children []dbus.Variant
}

type dbusMenu struct {
	host *Host
}

func newDBusMenu(host *Host) *dbusMenu { return &dbusMenu{host: host} }

type menuProperties struct{}

func newMenuProperties() *menuProperties { return &menuProperties{} }

func (p *menuProperties) Get(iface, property string) (dbus.Variant, *dbus.Error) {
	if iface != menuInterface {
		return dbus.Variant{}, dbus.NewError("org.freedesktop.DBus.Error.UnknownInterface", []any{iface})
	}
	value, ok := menuProps()[property]
	if !ok {
		return dbus.Variant{}, dbus.NewError("org.freedesktop.DBus.Error.UnknownProperty", []any{property})
	}
	return value, nil
}

func (p *menuProperties) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	if iface != menuInterface {
		return nil, dbus.NewError("org.freedesktop.DBus.Error.UnknownInterface", []any{iface})
	}
	return menuProps(), nil
}

func (p *menuProperties) Set(iface, property string, value dbus.Variant) *dbus.Error {
	return dbus.NewError("org.freedesktop.DBus.Error.PropertyReadOnly", []any{property})
}

func menuProps() map[string]dbus.Variant {
	return map[string]dbus.Variant{
		"Version":       dbus.MakeVariant(uint32(3)),
		"TextDirection": dbus.MakeVariant("ltr"),
		"Status":        dbus.MakeVariant("normal"),
		"IconThemePath": dbus.MakeVariant([]string{brand.TrayThemeDir()}),
	}
}

func (m *dbusMenu) GetLayout(parent, depth int32, propertyNames []string) (uint32, menuNode, *dbus.Error) {
	return 1, m.root(), nil
}

func (m *dbusMenu) GetGroupProperties(ids []int32, names []string) ([]struct {
	ID    int32
	Props map[string]dbus.Variant
}, *dbus.Error) {
	var out []struct {
		ID    int32
		Props map[string]dbus.Variant
	}
	for _, node := range m.items() {
		out = append(out, struct {
			ID    int32
			Props map[string]dbus.Variant
		}{ID: node.ID, Props: node.Props})
	}
	return out, nil
}

func (m *dbusMenu) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	for _, node := range m.items() {
		if node.ID == id {
			if value, ok := node.Props[name]; ok {
				return value, nil
			}
		}
	}
	return dbus.MakeVariant(""), nil
}

func (m *dbusMenu) Event(id int32, eventID string, data dbus.Variant, timestamp uint32) *dbus.Error {
	if eventID != "clicked" || m.host == nil {
		return nil
	}
	switch id {
	case menuCaptureArea:
		invokeAfterMenu(m.host.OnCaptureArea)
	case menuCaptureWindow:
		invokeAfterMenu(m.host.OnCaptureWindow)
	case menuCaptureScreen:
		invokeAfterMenu(m.host.OnCaptureScreen)
	case menuOpen:
		if m.host.OnShowWindow != nil {
			m.host.OnShowWindow()
		}
	case menuQuit:
		if m.host.OnQuit != nil {
			m.host.OnQuit()
		}
	}
	return nil
}

func invokeAfterMenu(fn func()) {
	if fn == nil {
		return
	}
	time.AfterFunc(menuDismissWait, fn)
}

func (m *dbusMenu) EventGroup(events []struct {
	ID        int32
	EventID   string
	Data      dbus.Variant
	Timestamp uint32
}) ([]int32, *dbus.Error) {
	for _, event := range events {
		_ = m.Event(event.ID, event.EventID, event.Data, event.Timestamp)
	}
	return nil, nil
}

func (m *dbusMenu) AboutToShow(id int32) (bool, *dbus.Error) { return false, nil }

func (m *dbusMenu) AboutToShowGroup(ids []int32) (updatesNeeded, idErrors []int32, err *dbus.Error) {
	return nil, nil, nil
}

func (m *dbusMenu) root() menuNode {
	children := make([]dbus.Variant, 0, 6)
	for _, node := range m.items() {
		children = append(children, dbus.MakeVariant(node))
	}
	return menuNode{
		ID: 0,
		Props: map[string]dbus.Variant{
			"children-display": dbus.MakeVariant("submenu"),
		},
		Children: children,
	}
}

func (m *dbusMenu) items() []menuNode {
	return []menuNode{
		item(menuCaptureArea, i18n.T("Capture area")),
		{
			ID: menuCaptureWindow,
			Props: map[string]dbus.Variant{
				"type":    dbus.MakeVariant("standard"),
				"label":   dbus.MakeVariant(i18n.T("Capture window")),
				"enabled": dbus.MakeVariant(false),
				"visible": dbus.MakeVariant(true),
			},
		},
		item(menuCaptureScreen, i18n.T("Capture screen")),
		{ID: 4, Props: map[string]dbus.Variant{"type": dbus.MakeVariant("separator")}},
		item(menuOpen, i18n.T("Open Lamha")),
		item(menuQuit, i18n.T("Quit")),
	}
}

func item(id int32, label string) menuNode {
	return menuNode{
		ID: id,
		Props: map[string]dbus.Variant{
			"type":    dbus.MakeVariant("standard"),
			"label":   dbus.MakeVariant(label),
			"enabled": dbus.MakeVariant(true),
			"visible": dbus.MakeVariant(true),
		},
	}
}
