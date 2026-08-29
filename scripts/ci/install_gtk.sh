#!/usr/bin/env bash
set -euo pipefail

install_first_available() {
  local pkg
  for pkg in "$@"; do
    if apt-cache show "$pkg" >/dev/null 2>&1; then
      sudo apt-get install -y --no-install-recommends "$pkg"
      return 0
    fi
  done
  echo "none of the packages are available: $*" >&2
  return 1
}

sudo apt-get update
sudo apt-get install -y --no-install-recommends \
  gcc \
  pkg-config \
  file \
  desktop-file-utils \
  dpkg-dev \
  libgtk-4-1 \
  libgtk-4-bin \
  libgtk-4-common \
  libgtk-4-dev \
  libglib2.0-dev \
  libcairo2-dev \
  libpango1.0-dev \
  libgdk-pixbuf-2.0-dev \
  librsvg2-dev \
  libgraphene-1.0-dev \
  gir1.2-gtk-4.0 \
  fonts-noto-core

# Ubuntu 24.04 uses 1.0; 25.10+ prefers the 2.0 girepository packages.
install_first_available libgirepository1.0-dev libgirepository-2.0-dev

# linuxdeploy-plugin-gtk copies gdk-pixbuf loaders via gdk-pixbuf-query-loaders.
install_first_available libgdk-pixbuf2.0-bin libgdk-pixbuf-2.0-bin

# GTK 4.14 ships these as $libdir/gtk-4.0/*.so; 4.22 builds them into libgtk-4.
install_first_available libgtk-4-media-gstreamer || true
install_first_available libgtk-4-media-ffmpeg || true

# GTK 4.20+ no longer installs $libdir/gtk-4.0. The linuxdeploy GTK plugin still
# copies that path, so create an empty tree when the modules were folded in.
gtk4_libdir="$(pkg-config --variable=libdir gtk4 2>/dev/null || true)"
if [[ -n "${gtk4_libdir:-}" && ! -d "$gtk4_libdir/gtk-4.0" ]]; then
  sudo mkdir -p "$gtk4_libdir/gtk-4.0"
fi
