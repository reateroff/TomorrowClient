#!/usr/bin/env bash
# Downloads wintun.dll next to the built exe (build/bin). It is NOT committed
# to git.
#
# This is the only runtime file TomorrowClient still needs: the sing-box core
# is linked into the binary, but it loads wintun.dll at runtime to create the
# TUN adapter, and Windows only searches the application directory for it.
#
# Usage: bash scripts/fetch-wintun.sh
set -euo pipefail

WINTUN_VER="0.14.1"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/build/bin"
TMP="$(mktemp -d)"
mkdir -p "$BIN"
trap 'rm -rf "$TMP"' EXIT

echo ">> wintun v$WINTUN_VER"
curl -sL -o "$TMP/wintun.zip" "https://www.wintun.net/builds/wintun-$WINTUN_VER.zip"
unzip -oj "$TMP/wintun.zip" "wintun/bin/amd64/wintun.dll" -d "$BIN" >/dev/null

echo ">> done"
ls -1 "$BIN"
