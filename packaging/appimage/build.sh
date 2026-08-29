#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/../.." && pwd)"
dist="$root/dist"
appdir="$dist/Lamha.AppDir"
arch="$(uname -m)"

case "$arch" in
  x86_64|aarch64) ;;
  *)
    echo "AppImage packaging is only scripted for x86_64 and aarch64, not $arch." >&2
    exit 1
    ;;
esac

if ! command -v go >/dev/null; then
  echo "Go is required to build Lamha." >&2
  exit 1
fi
if ! command -v pkg-config >/dev/null || ! pkg-config --exists gtk4; then
  echo "GTK4 development files and pkg-config are required (pkg-config gtk4)." >&2
  exit 1
fi
if ! command -v curl >/dev/null; then
  echo "curl is required to download linuxdeploy." >&2
  exit 1
fi

mkdir -p "$root/bin" "$dist"
(
  cd "$root"
  go build -o "$root/bin/lamha" ./cmd/lamha
)

rm -rf "$appdir"
mkdir -p \
  "$appdir/usr/bin" \
  "$appdir/usr/share/applications" \
  "$appdir/usr/share/icons/hicolor/scalable/apps" \
  "$appdir/usr/share/metainfo"

cp "$root/bin/lamha" "$appdir/usr/bin/lamha"
cp "$root/data/io.github.lamha.Lamha.desktop" "$appdir/usr/share/applications/io.github.lamha.Lamha.desktop"
cp "$root/data/io.github.lamha.Lamha.desktop" "$appdir/io.github.lamha.Lamha.desktop"
cp "$root/data/io.github.lamha.Lamha.metainfo.xml" "$appdir/usr/share/metainfo/io.github.lamha.Lamha.metainfo.xml"
cp "$root/data/icons/hicolor/scalable/apps/io.github.lamha.Lamha.svg" \
  "$appdir/usr/share/icons/hicolor/scalable/apps/io.github.lamha.Lamha.svg"
cp "$root/data/icons/hicolor/scalable/apps/io.github.lamha.Lamha.svg" \
  "$appdir/io.github.lamha.Lamha.svg"

linuxdeploy="$dist/linuxdeploy-${arch}.AppImage"
plugin="$dist/linuxdeploy-plugin-gtk.sh"
linuxdeploy_url="https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/linuxdeploy-${arch}.AppImage"
plugin_url="https://raw.githubusercontent.com/linuxdeploy/linuxdeploy-plugin-gtk/master/linuxdeploy-plugin-gtk.sh"

if [[ ! -x "$linuxdeploy" ]]; then
  curl -fsSL -o "$linuxdeploy" "$linuxdeploy_url"
  chmod +x "$linuxdeploy"
fi
if [[ ! -x "$plugin" ]]; then
  curl -fsSL -o "$plugin" "$plugin_url"
  chmod +x "$plugin"
fi

export APPIMAGE_EXTRACT_AND_RUN=1
export DEPLOY_GTK_VERSION=4
export PATH="$(dirname "$plugin"):$PATH"

cd "$dist"
"$linuxdeploy" \
  --appdir "$appdir" \
  --executable "$appdir/usr/bin/lamha" \
  --desktop-file "$appdir/usr/share/applications/io.github.lamha.Lamha.desktop" \
  --icon-file "$appdir/usr/share/icons/hicolor/scalable/apps/io.github.lamha.Lamha.svg" \
  --plugin gtk \
  --output appimage

echo "AppImage written under $dist"
