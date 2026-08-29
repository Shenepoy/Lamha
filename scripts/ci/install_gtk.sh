#!/usr/bin/env bash
set -euo pipefail

sudo apt-get update
sudo apt-get install -y --no-install-recommends \
  gcc \
  pkg-config \
  file \
  desktop-file-utils \
  libgtk-4-dev \
  libgirepository1.0-dev \
  libglib2.0-dev \
  libcairo2-dev \
  libpango1.0-dev \
  libgdk-pixbuf-2.0-dev \
  librsvg2-dev \
  libgraphene-1.0-dev \
  gir1.2-gtk-4.0 \
  fonts-noto-core
