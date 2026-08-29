#!/usr/bin/env bash
# Print name=YY.0M.MICRO from the VERSION file (Janan-style CalVer).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
name="$(tr -d '[:space:]' < "$ROOT_DIR/VERSION")"
if [[ ! "$name" =~ ^[0-9]{2}\.[0-9]{2}\.[0-9]+$ ]]; then
  echo "VERSION must be YY.0M.MICRO, got: ${name:-<empty>}" >&2
  exit 1
fi
echo "name=$name"
