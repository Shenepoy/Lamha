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

version="${LAMHA_VERSION:-}"
if [[ -z "$version" ]]; then
  version="$(tr -d '[:space:]' < "$root/VERSION")"
fi
if [[ ! "$version" =~ ^[0-9]{2}\.[0-9]{2}\.[0-9]+$ ]]; then
  echo "VERSION must be YY.0M.MICRO, got: ${version:-<empty>}" >&2
  exit 1
fi

ldflags="-s -w -X github.com/lamha-app/lamha/internal/version.Version=${version}"

mkdir -p "$root/bin" "$dist"
(
  cd "$root"
  go build -trimpath -ldflags "$ldflags" -o "$root/bin/lamha" ./cmd/lamha
)

rm -rf "$appdir"
mkdir -p \
  "$appdir/usr/bin" \
  "$appdir/usr/share/applications" \
  "$appdir/usr/share/icons/hicolor/scalable/apps" \
  "$appdir/usr/share/metainfo"

stamp_desktop() {
  local dest="$1"
  cp "$root/data/io.github.lamha.Lamha.desktop" "$dest"
  if ! grep -q '^X-AppImage-Version=' "$dest"; then
    printf '\nX-AppImage-Version=%s\n' "$version" >> "$dest"
  fi
}

cp "$root/bin/lamha" "$appdir/usr/bin/lamha"
stamp_desktop "$appdir/usr/share/applications/io.github.lamha.Lamha.desktop"
stamp_desktop "$appdir/io.github.lamha.Lamha.desktop"
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
export UPDATE_INFORMATION="gh-releases-zsync|Zyzto|Lamha|latest|Lamha-*x86_64.AppImage.zsync"
export LDAI_UPDATE_INFORMATION="$UPDATE_INFORMATION"
export PATH="$(dirname "$plugin"):$PATH"

# linuxdeploy-plugin-gtk copies $libdir/gtk-4.0. GTK 4.20+ may omit that tree.
gtk4_libdir="$(pkg-config --variable=libdir gtk4 2>/dev/null || true)"
if [[ -n "$gtk4_libdir" ]]; then
  export LD_GTK_LIBRARY_PATH="${LD_GTK_LIBRARY_PATH:-$gtk4_libdir}"
  if [[ ! -d "$gtk4_libdir/gtk-4.0" ]]; then
    mkdir -p "$gtk4_libdir/gtk-4.0" 2>/dev/null || sudo mkdir -p "$gtk4_libdir/gtk-4.0"
  fi
fi

cd "$dist"
"$linuxdeploy" \
  --appdir "$appdir" \
  --executable "$appdir/usr/bin/lamha" \
  --desktop-file "$appdir/usr/share/applications/io.github.lamha.Lamha.desktop" \
  --icon-file "$appdir/usr/share/icons/hicolor/scalable/apps/io.github.lamha.Lamha.svg" \
  --plugin gtk \
  --output appimage

bundle="$dist/Lamha-${arch}.AppImage"
if [[ ! -f "$bundle" ]]; then
  bundle="$(find "$dist" -maxdepth 1 -type f -name '*.AppImage' ! -name 'linuxdeploy*' | head -n 1 || true)"
fi
if [[ -z "${bundle}" || ! -f "$bundle" ]]; then
  echo "linuxdeploy did not write an AppImage under $dist" >&2
  exit 1
fi
named="$dist/Lamha-${version}-${arch}.AppImage"
mv -f "$bundle" "$named"
if command -v zsyncmake >/dev/null; then
  zsyncmake -u "https://github.com/Zyzto/Lamha/releases/latest/download/Lamha-${version}-${arch}.AppImage" \
    -o "${named}.zsync" "$named"
  echo "zsync written to ${named}.zsync"
fi
echo "AppImage written to $named"
