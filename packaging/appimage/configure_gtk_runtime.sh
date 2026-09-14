#!/usr/bin/env bash
# Repair linuxdeploy's GTK hook for Wayland AppImages.
set -euo pipefail

appdir="${1:-}"
if [[ -z "$appdir" || ! -d "$appdir" ]]; then
  echo "usage: $0 <AppDir>" >&2
  exit 2
fi

hook="$appdir/apprun-hooks/linuxdeploy-plugin-gtk.sh"
if [[ ! -f "$hook" ]]; then
  echo "GTK AppRun hook not found: $hook" >&2
  exit 1
fi

# linuxdeploy launches the payload through an AppRun.wrapped symlink. That
# makes GNOME and other task-list implementations label the process
# "AppRun.wrapped" and prevents them from matching the desktop entry/icon.
# Keep the generated environment hook, but launch the real named executable.
apprun="$appdir/AppRun"
if [[ -f "$apprun" ]] && grep -q 'AppRun\.wrapped' "$apprun"; then
  apprun_tmp="$(mktemp "${apprun}.XXXXXX")"
  trap 'rm -f "${hook_tmp:-}" "${apprun_tmp:-}"' EXIT
  sed 's#exec "\$this_dir"/AppRun\.wrapped "\$@"#exec "\$this_dir"/usr/bin/lamha "\$@"#' \
    "$apprun" > "$apprun_tmp"
  if grep -q 'AppRun\.wrapped' "$apprun_tmp"; then
    echo "could not replace AppRun.wrapped in $apprun" >&2
    exit 1
  fi
  chmod --reference="$apprun" "$apprun_tmp"
  mv -f "$apprun_tmp" "$apprun"
  apprun_tmp=""
fi

# Older linuxdeploy-plugin-gtk releases force X11 because their generated
# hook does not provide xkeyboard-config data. Remove that override and add a
# session-aware fallback below. An explicit GDK_BACKEND remains authoritative.
hook_tmp="$(mktemp "${TMPDIR:-/tmp}/lamha-gtk-hook.XXXXXX")"
trap 'rm -f "${hook_tmp:-}" "${apprun_tmp:-}"' EXIT
awk '
  $1 == "export" && $2 ~ /^GDK_BACKEND=x11/ { next }
  { print }
' "$hook" > "$hook_tmp"
mv -f "$hook_tmp" "$hook"
trap - EXIT

find_xkb_root() {
  local candidate
  for candidate in \
    "${XKB_CONFIG_ROOT:-}" \
    "${LAMHA_XKB_CONFIG_ROOT:-}" \
    "/usr/share/X11/xkb" \
    "/usr/share/xkeyboard-config-2" \
    "/usr/local/share/X11/xkb"; do
    if [[ -n "$candidate" && -d "$candidate/rules" ]]; then
      printf '%s\n' "$candidate"
      return 0
    fi
  done
  return 1
}

xkb_source="$(find_xkb_root || true)"
if [[ -z "$xkb_source" ]]; then
  echo "xkeyboard-config data not found; cannot build a safe Wayland AppImage" >&2
  exit 1
fi
xkb_target="$appdir/usr/share/X11/xkb"
if [[ ! -d "$xkb_target/rules" ]]; then
  mkdir -p "$xkb_target"
  case "$xkb_source" in
    "$xkb_target"|"$xkb_target"/*) ;;
    # Dereference distro/Nix store symlinks so the AppImage does not carry
    # links back to an unavailable host path.
    *) cp -aL "$xkb_source/." "$xkb_target/" ;;
  esac
fi
if [[ ! -f "$xkb_target/rules/evdev" || -L "$xkb_target/rules/evdev" ]]; then
  echo "xkeyboard-config data was not copied into $xkb_target" >&2
  exit 1
fi

cat >> "$hook" <<'EOF'

# Prefer native Wayland in a Wayland session. Keep explicit user overrides and
# retain X11 as a fallback for older desktops or X11-only sessions.
if [ -z "${GDK_BACKEND:-}" ]; then
    if [ -n "${WAYLAND_DISPLAY:-}" ] && [ "${XDG_SESSION_TYPE:-}" != "x11" ]; then
        export GDK_BACKEND=wayland
    else
        export GDK_BACKEND=x11
    fi
fi

# Conda-built libxkbcommon can retain its build-time xkeyboard-config path.
# Point it at the bundled data first, then use the host path when available.
if [ -z "${XKB_CONFIG_ROOT:-}" ] || [ ! -d "${XKB_CONFIG_ROOT}/rules" ]; then
    if [ -d "$APPDIR/usr/share/X11/xkb/rules" ]; then
        export XKB_CONFIG_ROOT="$APPDIR/usr/share/X11/xkb"
    elif [ -d "/usr/share/X11/xkb/rules" ]; then
        export XKB_CONFIG_ROOT="/usr/share/X11/xkb"
    fi
fi
EOF
