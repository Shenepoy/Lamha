#!/usr/bin/env bash
# Fail if an ELF needs a newer glibc than the portable baseline (Ubuntu 22.04).
set -euo pipefail

max="${LAMHA_MAX_GLIBC:-2.35}"
file="${1:-}"
if [[ -z "$file" || ! -f "$file" ]]; then
  echo "usage: $0 <elf>" >&2
  exit 1
fi
if ! command -v objdump >/dev/null; then
  echo "objdump is required (binutils)" >&2
  exit 1
fi

need="$(objdump -T "$file" 2>/dev/null | grep -oE 'GLIBC_[0-9]+\.[0-9]+' | sed 's/^GLIBC_//' | sort -uV | tail -n 1 || true)"
if [[ -z "$need" ]]; then
  echo "$file: no GLIBC version symbols"
  exit 0
fi

newest="$(printf '%s\n%s\n' "$need" "$max" | sort -V | tail -n 1)"
echo "$file: GLIBC_$need (limit GLIBC_$max)"
if [[ "$newest" != "$max" && "$need" != "$max" ]]; then
  echo "$file requires GLIBC_$need, newer than $max" >&2
  exit 1
fi
