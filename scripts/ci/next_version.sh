#!/usr/bin/env bash
# Next calendar version from VERSION and the current UTC month.
# First release in a month is YY.0M.0; later ones bump MICRO.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
current="$(tr -d '[:space:]' < "$ROOT_DIR/VERSION")"
year="$(date -u +%y)"
month="$(date -u +%m)"
prefix="${year}.${month}"
if [[ "$current" == "${prefix}."* ]]; then
  micro="${current##*.}"
  echo "${prefix}.$((micro + 1))"
else
  echo "${prefix}.0"
fi
