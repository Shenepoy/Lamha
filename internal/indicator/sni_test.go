package indicator

import (
	"errors"
	"testing"

	"github.com/godbus/dbus/v5"
)

type watcherCall struct {
	method string
	flags  dbus.Flags
	args   []interface{}
}

type fakeWatcher struct {
	calls []watcherCall
}

func (f *fakeWatcher) Call(method string, flags dbus.Flags, args ...interface{}) *dbus.Call {
	f.calls = append(f.calls, watcherCall{method: method, flags: flags, args: args})
	if len(args) != 1 {
		return &dbus.Call{Err: errors.New("RegisterStatusNotifierItem expects one argument")}
	}
	return &dbus.Call{}
}

func TestRegisterStatusNotifierItemUsesSingleServiceArgument(t *testing.T) {
	watcher := &fakeWatcher{}
	if err := registerStatusNotifierItem(watcher, "org.kde.StatusNotifierItem-test-1"); err != nil {
		t.Fatalf("registerStatusNotifierItem() error = %v", err)
	}
	if len(watcher.calls) != 1 {
		t.Fatalf("RegisterStatusNotifierItem calls = %d, want 1", len(watcher.calls))
	}
	call := watcher.calls[0]
	if call.method != watcherIface+".RegisterStatusNotifierItem" {
		t.Fatalf("method = %q", call.method)
	}
	if len(call.args) != 1 || call.args[0] != "org.kde.StatusNotifierItem-test-1" {
		t.Fatalf("arguments = %#v, want one service name", call.args)
	}
}
