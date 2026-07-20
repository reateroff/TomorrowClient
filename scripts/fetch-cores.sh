#!/usr/bin/env bash
# Downloads the external core binaries TomorrowClient needs at runtime and
# drops them next to the built exe (build/bin). These are NOT committed to git.
#
#   sing-box.exe  — default core (native TUN + auto_route)
#   xray.exe      — alternative core
#   tun2socks.exe — WinTun forwarder used by the xray core
#
# Usage: bash scripts/fetch-cores.sh
set -euo pipefail

SINGBOX_VER="1.11.15"
XRAY_VER="25.1.30"
T2S_VER="2.6.0"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/build/bin"
TMP="$(mktemp -d)"
mkdir -p "$BIN"
trap 'rm -rf "$TMP"' EXIT

echo ">> sing-box v$SINGBOX_VER"
curl -sL -o "$TMP/sb.zip" \
  "https://github.com/SagerNet/sing-box/releases/download/v$SINGBOX_VER/sing-box-$SINGBOX_VER-windows-amd64.zip"
unzip -oj "$TMP/sb.zip" "*/sing-box.exe" -d "$BIN" >/dev/null

echo ">> xray v$XRAY_VER"
curl -sL -o "$TMP/xray.zip" \
  "https://github.com/XTLS/Xray-core/releases/download/v$XRAY_VER/Xray-windows-64.zip"
unzip -oj "$TMP/xray.zip" "xray.exe" -d "$BIN" >/dev/null

echo ">> tun2socks v$T2S_VER"
curl -sL -o "$TMP/t2s.zip" \
  "https://github.com/xjasonlyu/tun2socks/releases/download/v$T2S_VER/tun2socks-windows-amd64.zip"
unzip -oj "$TMP/t2s.zip" "tun2socks-windows-amd64.exe" -d "$BIN" >/dev/null
mv -f "$BIN/tun2socks-windows-amd64.exe" "$BIN/tun2socks.exe"

echo ">> done. cores are in build/bin/"
ls -1 "$BIN"
