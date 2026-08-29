#!/usr/bin/env bash
set -euo pipefail

sudo apt-get update
sudo apt-get install -y --no-install-recommends \
  gcc \
  pkg-config \
  file \
  desktop-file-utils \
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
if apt-cache show libgirepository1.0-dev >/dev/null 2>&1; then
  sudo apt-get install -y --no-install-recommends libgirepository1.0-dev
else
  sudo apt-get install -y --no-install-recommends libgirepository-2.0-dev
fi
