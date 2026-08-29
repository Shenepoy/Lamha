package indicator

import (
	"fmt"
	"log"
	"os"

	"github.com/godbus/dbus/v5"

	"github.com/lamha-app/lamha/internal/brand"
)

const (
	sniInterface   = "org.kde.StatusNotifierItem"
	menuInterface  = "com.canonical.dbusmenu"
	sniObjectPath  = "/StatusNotifierItem"
	menuObjectPath = "/MenuBar"
	watcherKDE     = "org.kde.StatusNotifierWatcher"
	watcherPath    = "/StatusNotifierWatcher"
	watcherIface   = "org.kde.StatusNotifierWatcher"
)

// Host receives tray activations.
type Host struct {
	OnShowWindow    func()
	OnCaptureArea   func()
	OnCaptureWindow func()
	OnCaptureScreen func()
	OnQuit          func()
}

// Item is a StatusNotifierItem with a DBusMenu context menu.
type Item struct {
	host *Host
	conn *dbus.Conn
	menu *dbusMenu
}

// Start registers a background app indicator.
func Start(host *Host) (*Item, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect to session bus: %w", err)
	}

	item := &Item{host: host, conn: conn, menu: newDBusMenu(host)}
	name := fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
	reply, err := conn.RequestName(name, dbus.NameFlagDoNotQueue)
	if err != nil {
		return nil, fmt.Errorf("claim tray name: %w", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return nil, fmt.Errorf("tray name %s is already taken", name)
	}

	if err := conn.Export(item, sniObjectPath, sniInterface); err != nil {
		return nil, fmt.Errorf("export tray item: %w", err)
	}
	if err := conn.Export(newSNIProperties(item), sniObjectPath, "org.freedesktop.DBus.Properties"); err != nil {
		return nil, fmt.Errorf("export tray properties: %w", err)
	}
	if err := conn.Export(item.menu, menuObjectPath, menuInterface); err != nil {
		return nil, fmt.Errorf("export tray menu: %w", err)
	}
	if err := conn.Export(newMenuProperties(), menuObjectPath, "org.freedesktop.DBus.Properties"); err != nil {
		return nil, fmt.Errorf("export tray menu properties: %w", err)
	}

	if err := registerItem(conn, name); err != nil {
		return nil, err
	}

	log.Printf("app indicator registered as %s", name)
	return item, nil
}

func (i *Item) ContextMenu(x, y int32) *dbus.Error { return nil }

func (i *Item) Activate(x, y int32) *dbus.Error {
	if i.host != nil && i.host.OnShowWindow != nil {
		i.host.OnShowWindow()
	}
	return nil
}

func (i *Item) SecondaryActivate(x, y int32) *dbus.Error {
	return i.Activate(x, y)
}

func (i *Item) Scroll(delta int32, orientation string) *dbus.Error { return nil }

func (i *Item) properties() map[string]dbus.Variant {
	return map[string]dbus.Variant{
		"Category":      dbus.MakeVariant("ApplicationStatus"),
		"Id":            dbus.MakeVariant(brand.Name),
		"Title":         dbus.MakeVariant("Lamha"),
		"Status":        dbus.MakeVariant("Active"),
		"WindowId":      dbus.MakeVariant(uint32(0)),
		"IconName":      dbus.MakeVariant(brand.PanelName),
		"IconPixmap":    dbus.MakeVariant(brand.Pixmaps()),
		"ItemIsMenu":    dbus.MakeVariant(false),
		"Menu":          dbus.MakeVariant(dbus.ObjectPath(menuObjectPath)),
		"IconThemePath": dbus.MakeVariant(brand.TrayThemeDir()),
		"ToolTip":       dbus.MakeVariant(sniTooltip{brand.PanelName, brand.Pixmaps(), "Lamha", "Screenshot capture"}),
	}
}

type sniTooltip struct {
	Icon   string
	Pixmap []brand.Pixmap
	Title       string
	Description string
}

func registerItem(conn *dbus.Conn, name string) error {
	watchers := []string{watcherKDE, "org.freedesktop.StatusNotifierWatcher"}
	var last error
	for _, watcher := range watchers {
		call := conn.Object(watcher, watcherPath).Call(watcherIface+".RegisterStatusNotifierItem", 0, name)
		if call.Err == nil {
			return nil
		}
		last = call.Err
		call = conn.Object(watcher, watcherPath).Call(watcherIface+".RegisterStatusNotifierItem", 0, sniObjectPath)
		if call.Err == nil {
			return nil
		}
		last = call.Err
	}
	return fmt.Errorf("register tray item: %w", last)
}

type sniProperties struct {
	item *Item
}

func newSNIProperties(item *Item) *sniProperties { return &sniProperties{item: item} }

func (p *sniProperties) Get(iface, property string) (dbus.Variant, *dbus.Error) {
	if iface != sniInterface {
		return dbus.Variant{}, dbus.NewError("org.freedesktop.DBus.Error.UnknownInterface", []any{iface})
	}
	value, ok := p.item.properties()[property]
	if !ok {
		return dbus.Variant{}, dbus.NewError("org.freedesktop.DBus.Error.UnknownProperty", []any{property})
	}
	return value, nil
}

func (p *sniProperties) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	if iface != sniInterface {
		return nil, dbus.NewError("org.freedesktop.DBus.Error.UnknownInterface", []any{iface})
	}
	return p.item.properties(), nil
}

func (p *sniProperties) Set(iface, property string, value dbus.Variant) *dbus.Error {
	return dbus.NewError("org.freedesktop.DBus.Error.PropertyReadOnly", []any{property})
}
