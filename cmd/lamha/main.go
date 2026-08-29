package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/brand"
	"github.com/lamha-app/lamha/internal/hotkeys"
	"github.com/lamha-app/lamha/internal/i18n"
	"github.com/lamha-app/lamha/internal/indicator"
	"github.com/lamha-app/lamha/internal/keys"
	"github.com/lamha-app/lamha/internal/portal"
	"github.com/lamha-app/lamha/internal/prefs"
	"github.com/lamha-app/lamha/internal/ui"
	"github.com/lamha-app/lamha/internal/version"
)

const appID = "io.github.lamha.Lamha"

func main() {
	if versionRequested(os.Args[1:]) {
		fmt.Printf("lamha %s\n", version.String())
		return
	}

	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("lamha: ")
	log.Printf("starting pid=%d", os.Getpid())
	keys.Load()
	prefs.Load()
	i18n.SetLanguage(prefs.Current().Language())
	if err := brand.Install(); err != nil {
		log.Printf("could not install app icon: %v", err)
	}

	app := gtk.NewApplication(appID, gio.ApplicationHandlesCommandLine)
	app.AddMainOption(
		"capture",
		'c',
		glib.OptionFlagNone,
		glib.OptionArgString,
		i18n.T("Capture immediately (area, window, or screen)"),
		"MODE",
	)
	app.AddMainOption(
		"background",
		'b',
		glib.OptionFlagNone,
		glib.OptionArgNone,
		i18n.T("Start hidden in the background"),
		"",
	)

	var (
		window           *ui.Window
		servicesStarted  bool
		held             bool
		suppressActivate bool
	)

	ensureWindow := func() (*ui.Window, error) {
		if window != nil {
			return window, nil
		}
		created, err := ui.New(app)
		if err != nil {
			return nil, err
		}
		window = created
		return window, nil
	}

	hold := func() {
		if held {
			return
		}
		app.Hold()
		held = true
	}

	startServices := func(win *ui.Window) {
		if servicesStarted {
			return
		}
		servicesStarted = true
		hold()

		if _, err := indicator.Start(&indicator.Host{
			OnShowWindow:    func() { glib.IdleAdd(func() { win.Present() }) },
			OnCaptureArea:   func() { glib.IdleAdd(func() { win.StartCapture(ui.CaptureArea) }) },
			OnCaptureWindow: func() { glib.IdleAdd(func() { win.StartWindowPick() }) },
			OnCaptureScreen: func() { glib.IdleAdd(func() { win.StartCapture(ui.CaptureScreen) }) },
			OnQuit:          func() { glib.IdleAdd(func() { app.Quit() }) },
		}); err != nil {
			log.Printf("app indicator unavailable: %v", err)
		}

		if desktopIsGNOME() {
			if err := hotkeys.InstallGNOME(keys.Current().GNOMEAccels()); err != nil {
				log.Printf("system keybindings: %v", err)
			} else {
				log.Printf("system keybindings installed")
			}
		} else {
			go bindPortalShortcuts(win)
		}
		go requestBackgroundStay()
	}

	addAction := func(name string, fn func()) {
		action := gio.NewSimpleAction(name, nil)
		action.ConnectActivate(func(*glib.Variant) { fn() })
		app.AddAction(action)
	}

	app.ConnectStartup(func() {
		addAction("capture-area", func() {
			win, err := ensureWindow()
			if err != nil {
				return
			}
			startServices(win)
			win.StartCapture(ui.CaptureArea)
		})
		addAction("capture-window", func() {
			win, err := ensureWindow()
			if err != nil {
				return
			}
			startServices(win)
			win.StartCapture(ui.CaptureWindow)
		})
		addAction("capture-screen", func() {
			win, err := ensureWindow()
			if err != nil {
				return
			}
			startServices(win)
			win.StartCapture(ui.CaptureScreen)
		})
		addAction("show", func() {
			win, err := ensureWindow()
			if err != nil {
				return
			}
			startServices(win)
			win.Present()
		})
		addAction("quit", func() { app.Quit() })
	})

	app.ConnectActivate(func() {
		if suppressActivate {
			return
		}
		win, err := ensureWindow()
		if err != nil {
			log.Printf("starting Lamha: %v", err)
			return
		}
		startServices(win)
		win.Present()
	})

	app.ConnectCommandLine(func(commandLine *gio.ApplicationCommandLine) int {
		suppressActivate = true
		glib.IdleAdd(func() { suppressActivate = false })

		win, err := ensureWindow()
		if err != nil {
			log.Printf("starting Lamha: %v", err)
			commandLine.PrinterrLiteral(fmt.Sprintf("starting Lamha: %v\n", err))
			return 1
		}
		startServices(win)

		dict := commandLine.OptionsDict()
		if dict != nil && dict.Contains("capture") {
			value := dict.LookupValue("capture", glib.NewVariantType("s"))
			if value == nil {
				commandLine.PrinterrLiteral("missing capture mode\n")
				return 1
			}
			mode, err := ui.ParseCaptureMode(value.String())
			if err != nil {
				commandLine.PrinterrLiteral(err.Error() + "\n")
				return 1
			}
			win.StartCapture(mode)
			return 0
		}

		if dict != nil && dict.Contains("background") {
			return 0
		}

		win.Present()
		return 0
	})

	os.Exit(app.Run(os.Args))
}

func versionRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == "--version" || arg == "-V" {
			return true
		}
	}
	return false
}

func desktopIsGNOME() bool {
	return strings.Contains(strings.ToUpper(os.Getenv("XDG_CURRENT_DESKTOP")), "GNOME")
}

func currentPortalShortcuts() []portal.Shortcut {
	store := keys.Current()
	return []portal.Shortcut{
		{ID: portal.ShortcutArea, Description: i18n.T("Capture area"), Trigger: keys.ToPortalTrigger(store.Accel(keys.CaptureArea))},
		{ID: portal.ShortcutScreen, Description: i18n.T("Capture screen"), Trigger: keys.ToPortalTrigger(store.Accel(keys.CaptureScreen))},
	}
}

func bindPortalShortcuts(win *ui.Window) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	err := portal.NewClient().BindGlobalShortcuts(ctx, "", currentPortalShortcuts(), func(id string) {
		glib.IdleAdd(func() {
			switch id {
			case portal.ShortcutArea:
				win.StartCapture(ui.CaptureArea)
			case portal.ShortcutWindow:
				win.StartCapture(ui.CaptureWindow)
			case portal.ShortcutScreen:
				win.StartCapture(ui.CaptureScreen)
			}
		})
	})
	if err != nil {
		log.Printf("global shortcuts portal: %v", err)
		glib.IdleAdd(func() {
			if err := hotkeys.InstallGNOME(keys.Current().GNOMEAccels()); err != nil {
				log.Printf("system keybindings: %v", err)
			}
		})
	}
}

func requestBackgroundStay() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	// An empty parent avoids a second Wayland handle export racing capture.
	if err := portal.NewClient().RequestBackground(ctx, ""); err != nil {
		log.Printf("background stay: %v", err)
	}
}
