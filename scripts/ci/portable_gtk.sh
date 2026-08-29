#!/usr/bin/env bash
# Install a conda-forge GTK4 prefix so the AppImage can bundle modern GTK
# while the binary still links against an old glibc (Ubuntu 22.04 / 2.35).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PREFIX="${LAMHA_GTK_PREFIX:-$ROOT_DIR/.ci-gtk}"
SPEC="$ROOT_DIR/scripts/ci/portable_gtk.txt"
MAMBA_BIN="${LAMHA_MICROMAMBA:-$HOME/.local/bin/micromamba}"

export_var() {
  local name="$1"
  local value="$2"
  export "${name}=${value}"
  if [[ -n "${GITHUB_ENV:-}" ]]; then
    echo "${name}=${value}" >> "$GITHUB_ENV"
  fi
}

if [[ ! -x "$MAMBA_BIN" ]]; then
  tmp="$(mktemp -d)"
  curl -fsSL https://micro.mamba.pm/api/micromamba/linux-64/latest | tar -xj -C "$tmp" bin/micromamba
  mkdir -p "$(dirname "$MAMBA_BIN")"
  mv "$tmp/bin/micromamba" "$MAMBA_BIN"
  rm -rf "$tmp"
fi

if [[ ! -x "$PREFIX/bin/pkg-config" ]]; then
  mapfile -t specs < <(grep -vE '^[[:space:]]*(#|$)' "$SPEC")
  "$MAMBA_BIN" create -y -p "$PREFIX" -c conda-forge "${specs[@]}"
fi

pc_path="$PREFIX/lib/pkgconfig:${PREFIX}/share/pkgconfig"
export_var LAMHA_GTK_PREFIX "$PREFIX"
export_var PKG_CONFIG_PATH "$pc_path"
export_var PKG_CONFIG_LIBDIR "$pc_path"
export_var LD_LIBRARY_PATH "$PREFIX/lib${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
export_var LD_GTK_LIBRARY_PATH "$PREFIX/lib"
export_var GI_TYPELIB_PATH "$PREFIX/lib/girepository-1.0"
export_var XDG_DATA_DIRS "$PREFIX/share${XDG_DATA_DIRS:+:$XDG_DATA_DIRS}"
export_var PATH "$PREFIX/bin:$PATH"

pc="$PREFIX/bin/pkg-config"
if [[ ! -x "$pc" ]]; then
  pc="$PREFIX/bin/pkgconf"
fi
if ! "$pc" --exists --print-errors gtk4 glib-2.0; then
  echo "conda GTK prefix is missing gtk4 or glib-2.0 pkg-config files" >&2
  echo "PKG_CONFIG_PATH=$PKG_CONFIG_PATH" >&2
  find "$PREFIX" -name '*.pc' -print >&2 || true
  exit 1
fi

echo "gtk4 $($pc --modversion gtk4) glib $($pc --modversion glib-2.0) prefix=$PREFIX"
