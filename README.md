# Lamha

Lamha is a GTK4 screenshot application written in Go for Linux. It takes a
silent fullscreen snapshot through GNOME Shell, KWin, or Spectacle, then opens
Lamha's own overlay for selection and markup. It does not launch the GNOME or
KDE screenshot picker.

## What works now

- Freeze the screen and open Lamha's capture overlay
- Select an area, then annotate with pen, box, highlight, blur, numbered steps, and magic erase
- Delay a capture so menus and hover states can appear
- Hide the Lamha window before the shot so it is not included
- Save from the overlay or reopen **Annotate** later on a saved capture
- Copy to the clipboard when saving, if that option is checked
- Preview the latest image, browse local history, and copy to the clipboard
- Save captures in `$XDG_DATA_HOME/lamha/captures` (normally `~/.local/share/lamha/captures`)
- Start a capture from the command line or a desktop action
- Run on Wayland GNOME and KDE Plasma without compositor-specific shell commands

On GNOME the grab uses `org.gnome.Shell.Screenshot`. On Plasma it uses KWin or
a background Spectacle call. The XDG Screenshot portal is only a silent
fallback, never the interactive picker. The first time GNOME may ask for
screenshot permission; that is not the GNOME capture UI.

## Run it

GTK4 development headers and `pkg-config` are required. On NixOS (or with Nix
installed), the included shell supplies them:

```sh
nix develop
make run
```

Otherwise, install your distribution's GTK4 development package, `pkg-config`,
and Go, then run:

```sh
go run ./cmd/lamha
```

Capture without opening the window first:

```sh
lamha --capture=area
lamha --capture=window
lamha --capture=screen
```

If Lamha is already running, the same commands are forwarded to that instance.

Bind a desktop hotkey to one of those commands in GNOME Settings → Keyboard, or
KDE System Settings → Shortcuts. Right-clicking the application launcher also
exposes the same actions.

## AppImage

Build a portable bundle after installing Go, GTK4 development files, `pkg-config`,
and `curl`:

```sh
make appimage
```

The script downloads [linuxdeploy](https://github.com/linuxdeploy/linuxdeploy)
and its GTK plugin, then writes an AppImage under `dist/`. GTK4 bundling is most
reliable on a regular FHS distribution such as Fedora or Ubuntu. On NixOS, build
the AppImage from a container or another Linux system if linuxdeploy cannot
collect host libraries.

## Architecture

```text
GTK4 UI / CLI / desktop actions
        ↓
Silent grab (GNOME Shell / KWin / Spectacle)
        ↓
Lamha capture overlay (select + pen/box/highlight/blur/steps/erase)
        ↓
preview / clipboard / local history
```

This is intentionally a solid capture foundation rather than a claim to have
all of ShareX's feature set. The next useful milestones are upload destinations,
history search, and Portal Global Shortcuts.

## Verify

```sh
make fmt
make test
make build
```

For a user installation, install `bin/lamha` on `PATH`, copy
`data/io.github.lamha.Lamha.desktop` to `~/.local/share/applications/`, and copy
`data/icons/hicolor/scalable/apps/io.github.lamha.Lamha.svg` (from `Logo.svg`) to
`~/.local/share/icons/hicolor/scalable/apps/`.
