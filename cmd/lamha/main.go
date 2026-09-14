package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/brand"
	"github.com/lamha-app/lamha/internal/crash"
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
	// Keep os.Exit outside run so crash tracking defers can flush and close.
	os.Exit(run())
}

func run() (exitCode int) {
	if versionRequested(os.Args[1:]) {
		fmt.Printf("lamha %s\n", version.String())
		return 0
	}
	// AppImages launch through linuxdeploy's AppRun.wrapped symlink. Set the
	// GLib names explicitly so desktop shells use Lamha for the window/task
	// entry instead of exposing that implementation detail.
	glib.SetPrgname("lamha")
	glib.SetApplicationName("Lamha")

	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("lamha: ")
	crashSession, pendingCrash, crashErr := crash.Start(crash.AppInfo{
		Version: version.String(),
		PID:     os.Getpid(),
	})
	if crashErr != nil && !errors.Is(crashErr, crash.ErrAlreadyRunning) {
		log.Printf("crash tracking unavailable: %v", crashErr)
	}
	if crashSession != nil {
		log.SetOutput(io.MultiWriter(os.Stderr, crashSession.LogFile()))
	}
	defer func() {
		if value := recover(); value != nil {
			log.Printf("panic: %v", value)
			if crashSession != nil {
				crashSession.RecordPanic(value)
			}
			exitCode = 1
		}
		if crashSession != nil {
			_ = crashSession.Close()
		}
	}()
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
		tray             *indicator.Item
		servicesStarted  bool
		held             bool
		suppressActivate bool
		crashNoticeShown bool
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

	showPendingCrash := func(win *ui.Window) {
		if pendingCrash == nil || crashNoticeShown || win == nil {
			return
		}
		crashNoticeShown = true
		report := *pendingCrash
		glib.IdleAdd(func() {
			win.ShowCrashNotice(report, func() {
				if crashSession != nil {
					if err := crashSession.ClearPending(); err != nil {
						log.Printf("clearing crash report notice: %v", err)
					}
				}
				pendingCrash = nil
			})
		})
	}
	notifyPendingCrash := func(win *ui.Window) {
		if pendingCrash == nil || crashNoticeShown || win == nil {
			return
		}
		win.Notify(i18n.T("Lamha stopped unexpectedly"), i18n.T("Open Lamha to review the crash report."))
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

		var err error
		tray, err = indicator.Start(&indicator.Host{
			OnShowWindow:    func() { glib.IdleAdd(func() { win.Present(); showPendingCrash(win) }) },
			OnCaptureArea:   func() { glib.IdleAdd(func() { win.StartCapture(ui.CaptureArea) }) },
			OnCaptureWindow: func() { glib.IdleAdd(func() { win.StartWindowPick() }) },
			OnCaptureScreen: func() { glib.IdleAdd(func() { win.StartCapture(ui.CaptureScreen) }) },
			OnQuit:          func() { glib.IdleAdd(func() { app.Quit() }) },
		})
		if err != nil {
			log.Printf("app indicator unavailable: %v", err)
		} else if tray == nil {
			log.Printf("app indicator returned no item")
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
			notifyPendingCrash(win)
			win.StartCapture(ui.CaptureArea)
		})
		addAction("capture-window", func() {
			win, err := ensureWindow()
			if err != nil {
				return
			}
			startServices(win)
			notifyPendingCrash(win)
			win.StartCapture(ui.CaptureWindow)
		})
		addAction("capture-screen", func() {
			win, err := ensureWindow()
			if err != nil {
				return
			}
			startServices(win)
			notifyPendingCrash(win)
			win.StartCapture(ui.CaptureScreen)
		})
		addAction("show", func() {
			win, err := ensureWindow()
			if err != nil {
				return
			}
			startServices(win)
			win.Present()
			showPendingCrash(win)
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
		showPendingCrash(win)
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
			notifyPendingCrash(win)
			win.StartCapture(mode)
			return 0
		}

		if dict != nil && dict.Contains("background") {
			if pendingCrash != nil {
				win.Present()
				showPendingCrash(win)
			}
			return 0
		}

		win.Present()
		showPendingCrash(win)
		return 0
	})

	exitCode = app.Run(os.Args)
	return exitCode
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
