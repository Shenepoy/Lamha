#!/usr/bin/env bash
# Host toolchain for the portable AppImage job. Do not install distro GTK:
# pkg-config must see the conda-forge prefix from portable_gtk.sh.
set -euo pipefail

sudo apt-get update
sudo apt-get install -y --no-install-recommends \
  gcc \
  libc6-dev \
  curl \
  ca-certificates \
  file \
  desktop-file-utils \
  binutils \
  xz-utils \
  bzip2 \
  zsync
